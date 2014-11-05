// Package releaser determines vesicle release events and latencies for our
// from and mouse NMJ AZ models.
package releaser

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/haskelladdict/mbdr/libmbd"
)

// type of binding site (syt or second sensor)
const (
	SytSite = iota
	YSite
)

// AnalyzerInfo keeps basic stats on the analyzer itself
type AnalyzerInfo struct {
	Name       string
	Version    string
	NumThreads int
}

// SimModel encapsulates all information related to the simulation/model itself
// fusion events
type SimModel struct {
	CaSensors      []CaSensor        // list of Ca sensor sites per synaptotagmin/Y site
	VesicleIDs     []string          // list of vesicle IDs
	VGCCVesicleMap map[string]string // map of vesicles to their main channel
	SensorTemplate string            // fmt string the analyzer will use to extract binding events
	NumPulses      int               // number of stimulation events in data
	IsiValue       float64           // value of interstimulus interval
	PulseDuration  float64           // how long does a single pulse last
}

// FusionModel describes the basic ingredients of the fusion model
type FusionModel struct {
	NumSyt              int  // number of synaptotagmin molecules (with 5 Ca2+ sites each)
	NumY                int  // number of second sensor (Y) sites
	NumActiveSyt        int  // how many Ca2+ sites need to be bound for sensors
	NumActiveY          int  // to become active
	VesicleFusionEnergy int  // energy needed to fuse vesicle in energy model
	EnergyModel         bool // use the energy model
	SytEnergy           int  // energy of activated synaptotagmin toward vesicle fusion
	YEnergy             int  // energy of activated Y sites toward vesicle fusion
	NumActiveSites      int  // number of simultaneously active sites required for release
}

// CaSensor defines a single synaptotagmin and Y sites
type CaSensor struct {
	Sites    []int // ca sites contributing to sensors
	SiteType int   // type of sensor (syt or Y)
}

// ActEvent keeps track of a single activation/deactivation event
type ActEvent struct {
	sensorID  int    // sensor which was activated/deactivated
	vesicleID string // vesicleID were activation event took place
	eventIter int    // iteration when event occured
	activated bool   // activated is set to true and deactivated otherwise
}

// sort infrastructure for sorting ActEvents according to the event time
type byIter []ActEvent

func (e byIter) Len() int {
	return len(e)
}

func (e byIter) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e byIter) Less(i, j int) bool {
	return e[i].eventIter < e[j].eventIter
}

// ReleaseEvent keeps track of vesicle release events
type ReleaseEvent struct {
	sensors   []int  // list of sensors involved in release event
	vesicleID string // id of vesicle which was released
	eventIter uint64 // iteration when event occurred
}

// analyze is the main entry point for analyzing the mouse AZ model. It
// determines release events and collects statistics
func analyze(data *libmbd.MCellData, m *SimModel, fusion *FusionModel,
	rng *rand.Rand, seed int) ([]string, error) {

	var releases []*ReleaseEvent
	for _, vesID := range m.VesicleIDs {
		evts, err := extractActivationEvents(data, m, fusion, seed, vesID)
		if err != nil {
			return nil, err
		}
		if evts == nil {
			continue
		}

		rel, err := extractReleaseEvents(evts, m, fusion, data.BlockLen(), vesID, rng)
		if err != nil {
			return nil, err
		}
		if rel != nil {
			releases = append(releases, rel)
		}
	}

	return assembleReleaseMsgs(data, m, seed, releases), nil
}

// assembleReleaseMsgs creates a slice of strings with summary statistics for all
// released vesicles for a given seed
func assembleReleaseMsgs(data *libmbd.MCellData, m *SimModel, seed int,
	rel []*ReleaseEvent) []string {
	var messages []string
	timeStep := data.OutputStepLen()
	for _, r := range rel {
		buffer := bytes.NewBufferString("")
		channels, err := determineCaChanContrib(data, r)
		if err != nil {
			log.Fatal(err)
		}
		if err := checkCaNumbers(m.CaSensors, channels, r); err != nil {
			fmt.Printf("In seed %d, vesicle %s, time %f\n", seed, r.vesicleID,
				float64(r.eventIter)*data.OutputStepLen())
			log.Fatal(err)
		}

		eventTime := float64(r.eventIter) * timeStep
		pulseString := gatherPulseID(m.IsiValue, m.PulseDuration, eventTime)

		fmt.Fprintf(buffer, "seed : %d   vesicleID : %s   time : %e   pulseID : %s",
			seed, r.vesicleID, eventTime, pulseString)

		fmt.Fprintf(buffer, "  sensors : |")
		// sort sensors to make output consistent across runs
		var sensors = sort.IntSlice(r.sensors)
		sensors.Sort()
		for _, s := range sensors {
			fmt.Fprintf(buffer, "%d|", s)
		}

		chans, mainChan, totalCa := gatherVGCCData(m.VGCCVesicleMap,
			channels, r.vesicleID)
		fmt.Fprintf(buffer, "  channels : %s", chans)
		fmt.Fprintf(buffer, "  totalCaBound : %d", totalCa)
		fmt.Fprintf(buffer, "  mainChannelContrib : %s", mainChan)
		fmt.Fprintf(buffer, "  numContribChannels : %d", len(channels))

		messages = append(messages, buffer.String())
	}
	return messages
}

// gatherPulseID determines the pulse or interstimulus ID during which
// a release happened and then returns it as a string
func gatherPulseID(isi, duration, eventTime float64) string {
	// figure out if event happened within or between pulses
	var pulseID int
	if isi == 0 {
		pulseID = 0
	} else {
		pulseID = int(math.Floor(eventTime / isi))
	}
	var pulseString string
	if eventTime-float64(pulseID)*isi > duration {
		pulseString = fmt.Sprintf("ISI_%d", pulseID+1)
	} else {
		pulseString = fmt.Sprintf("%d", pulseID+1)
	}
	return pulseString
}

// gatherVGCCData gathers the IDs of the channels contributing to the release,
// if the main channel was involved in release and the total number of bound
// calcium ions
func gatherVGCCData(vesMap map[string]string, channels map[string]float64,
	vesicleID string) (string, string, int) {

	var totalCa int
	var mainChannel string
	if vesMap != nil {
		mainChannel = vesMap[vesicleID]
	}
	var haveMainChannel bool

	// sort channels to make output consistent across runs
	var cs sort.StringSlice
	for n := range channels {
		cs = append(cs, n)
	}
	cs.Sort()

	buffer := bytes.NewBufferString("")
	fmt.Fprintf(buffer, "|")
	for _, c := range cs {
		numCa := int(channels[c])
		totalCa += numCa
		if c == mainChannel {
			haveMainChannel = true
		}
		fmt.Fprintf(buffer, "%s:%d|", c, int(channels[c]))
	}

	// assemble indicator if main channel was involved in release or not. NA
	// indicates that we didn't have the VGCC-Vesicle mapping available
	mainChannelMsg := "NA"
	if haveMainChannel {
		mainChannelMsg = "Y"
	} else if mainChannel != "" {
		mainChannelMsg = "N"
	}
	return buffer.String(), mainChannelMsg, totalCa
}

// extractActivationEvents returns a slice with actvation and deactivation events
// for the given vesicle and active zone
func extractActivationEvents(data *libmbd.MCellData, m *SimModel, fusion *FusionModel,
	seed int, vesicleID string) ([]ActEvent, error) {

	var events []ActEvent
	// analyze the activation/deactivation status of each ca sensor.
	// NOTE: for now we merge the binding data for individual pulses into one
	for id := 0; id < len(m.CaSensors); id++ {
		sensor := m.CaSensors[id]
		sensorString := "sensor"
		actThresh := fusion.NumActiveSyt
		if sensor.SiteType == YSite {
			sensorString = "sensor_Y"
			actThresh = fusion.NumActiveY
		}

		// NOTE: This could be improved. the templates differ depending on if the
		// underlying data comes from a single or multi-pulse experiment
		var dataNames []string
		for _, s := range sensor.Sites {
			if m.NumPulses == 1 {
				dataNames = append(dataNames, fmt.Sprintf(m.SensorTemplate, vesicleID,
					sensorString, s, seed))
			} else {
				for p := 0; p < m.NumPulses; p++ {
					dataNames = append(dataNames, fmt.Sprintf(m.SensorTemplate, vesicleID,
						sensorString, s, p+1, seed))
				}
			}
		}

		sensorData := make([]int, data.BlockLen())
		for _, dataName := range dataNames {
			bd, err := data.BlockDataByName(dataName)
			if err != nil {
				return nil, err
			}

			if len(bd.Col) != 1 {
				return nil, fmt.Errorf("data set %s had more than one data column",
					dataName)
			}
			for i := 0; i < len(sensorData); i++ {
				sensorData[i] += int(bd.Col[0][i])
			}
		}

		// check for activation events
		active := false
		for i, b := range sensorData {
			if !active && b >= actThresh {
				active = true
				events = append(events, ActEvent{id, vesicleID, i, active})
			} else if active && b < actThresh {
				active = false
				events = append(events, ActEvent{id, vesicleID, i, active})
			}
		}
	}
	return events, nil
}

// extractReleaseEvents determines if the given vesicle was released given
// a list of sensor activation events. If no release took place returns nil.
func extractReleaseEvents(evts []ActEvent, model *SimModel, fusion *FusionModel,
	maxIter uint64, vesicleID string, rng *rand.Rand) (*ReleaseEvent, error) {

	sort.Sort(byIter(evts))
	activeEvts := make(map[int]struct{})
	for i, e := range evts {
		_, present := activeEvts[e.sensorID]
		if e.activated {
			if present {
				return nil, fmt.Errorf("trying to add active event that already exists")
			}
			activeEvts[e.sensorID] = struct{}{}
		} else {
			if !present {
				return nil, fmt.Errorf("trying to remove a nonexisting active event")
			}
			delete(activeEvts, e.sensorID)
		}

		// special case: If the next event happens simultaneously we apply it right away
		if i+1 < len(evts) && evts[i+1].eventIter == e.eventIter {
			continue
		}

		var rel *ReleaseEvent
		var relError error
		if fusion.EnergyModel {
			// use the energy model to determine release
			energy := getEnergy(model.CaSensors, activeEvts, fusion.SytEnergy, fusion.YEnergy)
			nextEvtIter := getNextEvtIter(i, maxIter, evts)
			rel, relError = checkForEnergyRelease(fusion.VesicleFusionEnergy, energy,
				vesicleID, e, activeEvts, nextEvtIter, rng)
		} else {
			// use the deterministic model to determine release
			rel, relError = checkForDeterministicRelease(vesicleID, fusion.NumActiveSites,
				e, activeEvts)
		}
		if relError != nil {
			return nil, relError
		}
		if rel != nil {
			return rel, nil
		}
	}
	return nil, nil
}

// getEnergy computes the total energy corresponding to the current number
// of active synaptotagmin and Y sites. Also returns the number of active syts
func getEnergy(caSensors []CaSensor, events map[int]struct{}, sytEnergy, yEnergy int) int {
	var energy int
	for s := range events {
		if caSensors[s].SiteType == SytSite {
			energy += sytEnergy
		} else {
			energy += yEnergy
		}
	}
	return energy
}

// getNextEvtIter determines the iteration of the next event that will happen in
// the event queue
func getNextEvtIter(iter int, maxIter uint64, evts []ActEvent) uint64 {
	nextIter := maxIter
	if iter < len(evts)-1 {
		nextIter = uint64(evts[iter+1].eventIter)
	}
	return nextIter
}

// checkForDeterministicRelease tests if vesicles are released according
// to a deterministic critertion, i.e. as soon as numActiveSites syt or
// Y sites are active
func checkForDeterministicRelease(vesID string, numActiveSites int, evt ActEvent,
	activeEvts map[int]struct{}) (*ReleaseEvent, error) {
	if len(activeEvts) == numActiveSites {
		var sensors []int
		for a := range activeEvts {
			sensors = append(sensors, a)
		}
		return &ReleaseEvent{sensors: sensors, vesicleID: vesID,
			eventIter: uint64(evt.eventIter)}, nil
	}
	return nil, nil
}

// checkForEnergyRelease tests if an energy release according to specified
// syt and y site energies takes place. Check for releases given the current
// energy until next event or the end of simulation. To do this we basically
// test for each iteration between now and the next event if a release takes
// place using the Metropolis-Hastings algorithm
func checkForEnergyRelease(fusionEnergy, energy int, vesID string, evt ActEvent,
	activeEvts map[int]struct{}, nextEvtIter uint64, rng *rand.Rand) (*ReleaseEvent, error) {

	numIters := nextEvtIter - uint64(evt.eventIter)
	if nextEvtIter < uint64(evt.eventIter) {
		return nil, fmt.Errorf("encountered out of order release event")
	}
	if iter, ok := checkForRelease(fusionEnergy, energy, numIters, rng); ok {
		var sensors []int
		for a := range activeEvts {
			sensors = append(sensors, a)
		}
		return &ReleaseEvent{sensors: sensors, vesicleID: vesID,
			eventIter: uint64(evt.eventIter) + iter}, nil
	}
	return nil, nil
}

// checkForReleases uses a Metropolis-Hasting scheme to test numIter times
// if vesicle release happens given the provided bound sensor energy
func checkForRelease(vesicleFusionEnergy, energy int, numIters uint64,
	rng *rand.Rand) (uint64, bool) {

	if energy >= vesicleFusionEnergy {
		return 0, true
	}

	prob := math.Exp(float64(energy - vesicleFusionEnergy))
	if prob >= 1 {
		log.Fatal("probability out of bounds")
	}
	for i := uint64(0); i < numIters; i++ {
		if rng.Float64() < prob {
			return i, true
		}
	}
	return 0, false
}

// checkCaNumbers does a sanity check to ensure that the number of bound
// calcium ions is equal or larger than what is expected based on the activated
// syt and Y sites
func checkCaNumbers(caSensors []CaSensor, channels map[string]float64, r *ReleaseEvent) error {
	var expected int
	for _, s := range r.sensors {
		if caSensors[s].SiteType == SytSite {
			expected += 2
		} else if caSensors[s].SiteType == YSite {
			expected++
		} else {
			return fmt.Errorf("in checkCaNumbers: Encountered incorrect binding site "+
				"type %d", caSensors[s].SiteType)
		}
	}

	var actual int
	for _, c := range channels {
		actual += int(c)
	}

	if actual < expected {
		return fmt.Errorf("The number of bound Ca ions (%d) is smaller than expected "+
			"based on the activation status (%d)", actual, expected)
	}
	return nil
}

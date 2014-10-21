package releaser

import (
	"bytes"
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
	"log"
	"math"
	"math/rand"
	"sort"
)

// type of binding site (syt or second sensor)
const (
	SytSite = iota
	YSite
)

// SimModel encapsulates all information related to the simulation/model itself
// fusion events
type SimModel struct {
	CaSensors      []CaSensor
	VesicleIDs     []string
	SensorTemplate string
	NumPulses      int
	IsiValue       float64
	PulseDuration  float64
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
	messages := make([]string, 0)
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
		// figure out if event happened within or between pulses
		var pulseID int
		if m.IsiValue == 0 {
			pulseID = 0
		} else {
			pulseID = int(math.Floor(eventTime / m.IsiValue))
		}
		var pulseString string
		if eventTime-float64(pulseID)*m.IsiValue > m.PulseDuration {
			pulseString = fmt.Sprintf("ISI_%d", pulseID+1)
		} else {
			pulseString = fmt.Sprintf("%d", pulseID+1)
		}

		fmt.Fprintf(buffer, "seed : %d   vesicleID : %s   time : %e   pulseID : %s", seed,
			r.vesicleID, eventTime, pulseString)
		fmt.Fprintf(buffer, "  sensors: [")
		for _, s := range r.sensors {
			fmt.Fprintf(buffer, "%d ", s)
		}
		fmt.Fprintf(buffer, "]")
		fmt.Fprintf(buffer, "  channels: [")
		for n, c := range channels {
			fmt.Fprintf(buffer, "%s : %d  ", n, int(c))
		}
		fmt.Fprintf(buffer, "]")
		messages = append(messages, buffer.String())
	}
	return messages
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
	for s, _ := range events {
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
		for a, _ := range activeEvts {
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
		for a, _ := range activeEvts {
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
			expected += 1
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

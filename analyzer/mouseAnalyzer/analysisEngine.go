package main

import (
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
	"log"
	"math"
	"math/rand"
	"sort"
	"time"
)

const (
	numAZ       = 6  // number of active zones (AZ) in the model
	numVesicles = 2  // number of vesicles per AZ
	numSyt      = 8  // number of synaptotagmin molecules (with 5 Ca2+ sites each)
	numY        = 16 // number of second sensor (Y) sites
)

const (
	numActiveSyt        = 2 // how many Ca2+ sites need to be bound for sensors
	numActiveY          = 1 // to become active
	vesicleFusionEnergy = 40
)

// type of binding site (syt or second sensor)
const (
	sytSite = iota
	ySite
)

// CaSensor defines a single synaptotagmin and Y sites
type caSensor struct {
	sites    []int // ca sites contributing to sensors
	siteType int   // type of sensor (syt or Y)
}

// ActEvent keeps track of a single activation/deactivation event
type ActEvent struct {
	sensorID  int  // sensor which was activated/deactivated
	azId      int  // id of vesicle and active zone where activation
	vesicleID int  // event took place
	eventIter int  // iteration when event occured
	activated bool // activated is set to true and deactivated otherwise
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

// RelEvent keeps track of vesicle release events
type RelEvent struct {
	sensors   []int  // list of sensors involved in release event
	azId      int    // id of vesicle and active zone where activation
	vesicleID int    // event took place
	eventIter uint64 // iteration when event occured
}

var caSensors []caSensor

// random number generator for Metropolis-Hasting
var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	caSensors = make([]caSensor, numSyt+numY)

	// define synaptogamin and Y sites
	caSensors[0] = caSensor{[]int{8, 9, 29, 30, 31}, sytSite}
	caSensors[1] = caSensor{[]int{7, 32, 33, 34, 35}, sytSite}
	caSensors[2] = caSensor{[]int{3, 6, 36, 37, 38}, sytSite}
	caSensors[3] = caSensor{[]int{17, 39, 40, 41, 42}, sytSite}
	caSensors[4] = caSensor{[]int{15, 16, 43, 44, 45}, sytSite}
	caSensors[5] = caSensor{[]int{14, 46, 47, 48, 49}, sytSite}
	caSensors[6] = caSensor{[]int{4, 12, 24, 50, 51}, sytSite}
	caSensors[7] = caSensor{[]int{10, 25, 26, 27, 28}, sytSite}
	caSensors[8] = caSensor{[]int{122}, ySite}
	caSensors[9] = caSensor{[]int{70}, ySite}
	caSensors[10] = caSensor{[]int{126}, ySite}
	caSensors[11] = caSensor{[]int{142}, ySite}
	caSensors[12] = caSensor{[]int{62}, ySite}
	caSensors[13] = caSensor{[]int{118}, ySite}
	caSensors[14] = caSensor{[]int{22}, ySite}
	caSensors[15] = caSensor{[]int{134}, ySite}
	caSensors[16] = caSensor{[]int{110}, ySite}
	caSensors[17] = caSensor{[]int{66}, ySite}
	caSensors[18] = caSensor{[]int{106}, ySite}
	caSensors[19] = caSensor{[]int{130}, ySite}
	caSensors[20] = caSensor{[]int{2}, ySite}
	caSensors[21] = caSensor{[]int{114}, ySite}
	caSensors[22] = caSensor{[]int{42}, ySite}
	caSensors[23] = caSensor{[]int{138}, ySite}
}

// analyze is the main entry point for analyzing the mouse AZ model. It
// determines release events and collects statistics
func analyze(data *libmbd.MCellData, seed int, numPulses, sytEnergy, yEnergy int) error {

	for az := 0; az < numAZ; az++ {
		for ves := 0; ves < numVesicles; ves++ {
			evts, err := extractActivationEvents(data, seed, az, ves)
			if err != nil {
				return err
			}
			if evts == nil {
				continue
			}

			rel, err := extractReleaseEvents(evts, data.BlockSize(), sytEnergy,
				yEnergy, az, ves)
			if err != nil {
				return err
			}
			if rel != nil {
				fmt.Println("found release ", rel)
			}
		}
	}
	return nil
}

// extractActivationEvents returns a slice with actvation and deactivation events
// for the given vesicle and active zone
func extractActivationEvents(data *libmbd.MCellData, seed, az, ves int) ([]ActEvent, error) {

	var events []ActEvent
	// analyze the activation/deactivation status of each ca sensor.
	// NOTE: for now we merge the binding data for individual pulses into one
	for id := 0; id < len(caSensors); id++ {
		sensor := caSensors[id]
		sensorString := "sensor"
		actThresh := numActiveSyt
		if sensor.siteType == ySite {
			sensorString = "sensor_Y"
			actThresh = numActiveY
		}

		sensorData := make([]int, data.BlockSize())
		for _, s := range sensor.sites {
			for p := 0; p < numPulsesFlag; p++ {
				dataName := fmt.Sprintf("bound_vesicle_%d_%d_%s_%d_%d.%04d.dat", az+1,
					ves+1, sensorString, s, p+1, seed)
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
		}

		// check for activation events
		active := false
		for i, b := range sensorData {
			if !active && b >= actThresh {
				active = true
				events = append(events, ActEvent{id, az, ves, i, active})
			} else if active && b < actThresh {
				active = false
				events = append(events, ActEvent{id, az, ves, i, active})
			}
		}
	}
	return events, nil
}

// extractReleaseEvents determines if the given vesicle was released given
// a list of sensor activation events. If no release took place returns nil.
func extractReleaseEvents(evts []ActEvent, maxIter uint64, sytEnergy, yEnergy,
	az, ves int) (*RelEvent, error) {

	if len(evts) == 0 {
		return nil, fmt.Errorf("attempted to analyze an empty event list")
	}

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

		// get current total energy
		var energy int
		for s, _ := range activeEvts {
			if caSensors[s].siteType == sytSite {
				energy += sytEnergy
			} else {
				energy += yEnergy
			}
		}

		// Now check for releases given the current energy until next event or
		// the end of simulation. To do this we basically test for each iteration
		// between now and the next event if a release takes place using the
		// Metrolpolis-Hasting algorithm
		nextIter := maxIter
		if i < len(evts)-1 {
			nextIter = uint64(evts[i+1].eventIter)
		}
		numIters := nextIter - uint64(e.eventIter)
		if nextIter < uint64(e.eventIter) {
			return nil, fmt.Errorf("encountered out of order release event")
		}
		if iter, ok := checkForRelease(energy, numIters); ok {
			var sensors []int
			for a, _ := range activeEvts {
				sensors = append(sensors, a)
			}
			return &RelEvent{
				sensors:   sensors,
				azId:      az,
				vesicleID: ves,
				eventIter: uint64(e.eventIter) + iter,
			}, nil
		}
	}
	return nil, nil
}

// checkForReleases uses a Metropolis-Hasting scheme to test numIter times
// if vesicle release happens given the provided bound sensor energy
func checkForRelease(energy int, numIters uint64) (uint64, bool) {

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

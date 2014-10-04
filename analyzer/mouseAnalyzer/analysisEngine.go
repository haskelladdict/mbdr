package main

import (
	"fmt"
	"github.com/haskelladdict/mbdr/libmbd"
)

const (
	numAZ       = 6  // number of active zones (AZ) in the model
	numVesicles = 2  // number of vesicles per AZ
	numSyt      = 8  // number of synaptotagmin molecules (with 5 Ca2+ sites each)
	numY        = 16 // number of second sensor (Y) sites
)

const vesicleFusionEnergy = 40

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
	sensor    *caSensor // sensor which was activated/deactivated
	azId      int       // id of vesicle and active zone where activation
	vesicleID int       // event took place
	activated bool      // activated is set to true and deactivated otherwise
}

var caSensors []caSensor

func init() {
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
	caSensors[8] = caSensor{[]int{70}, ySite}
	caSensors[8] = caSensor{[]int{126}, ySite}
	caSensors[8] = caSensor{[]int{142}, ySite}
	caSensors[8] = caSensor{[]int{62}, ySite}
	caSensors[8] = caSensor{[]int{118}, ySite}
	caSensors[8] = caSensor{[]int{22}, ySite}
	caSensors[8] = caSensor{[]int{134}, ySite}
	caSensors[8] = caSensor{[]int{110}, ySite}
	caSensors[8] = caSensor{[]int{66}, ySite}
	caSensors[8] = caSensor{[]int{106}, ySite}
	caSensors[8] = caSensor{[]int{130}, ySite}
	caSensors[8] = caSensor{[]int{2}, ySite}
	caSensors[8] = caSensor{[]int{114}, ySite}
	caSensors[8] = caSensor{[]int{42}, ySite}
	caSensors[8] = caSensor{[]int{138}, ySite}
}

// analyze is the main entry point for analyzing the mouse AZ model. It
// determines release events and collects statistics
func analyze(data *libmbd.MCellData, seed int, numPulses, sytEnergy, yEnergy int) error {

	for az := 0; az < numAZ; az++ {
		for ves := 0; ves < numVesicles; ves++ {
			_, err := extractActivationEvents(data, seed, az, ves)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("foo", len(caSensors))
	return nil
}

// extractActivationEvents returns a slice with actvation and deactivation events
// for the given vesicle and active zone
func extractActivationEvents(data *libmbd.MCellData, seed int, az,
	ves int) ([]ActEvent, error) {

	var events []ActEvent
	// analyze the activation/deactivation status of each ca sensor.
	// NOTE: for now we merge the binding data for individual pulses into one
	for _, sensor := range caSensors {
		sensorString := "sensor"
		if sensor.siteType == ySite {
			sensorString = "sensor_Y"
		}

		for _, s := range sensor.sites {
			for p := 0; p < numPulsesFlag; p++ {
				dataName := fmt.Sprintf("bound_vesicle_%d_%d_%s_%d_%d.%04d.dat", az, ves,
					sensorString, s, p, seed)
				fmt.Println(dataName)
			}
		}
	}
	return events, nil
}

/*
















*/

// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Bram Gruneir (bram+code@cockroachlabs.com)

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cockroachdb/cockroach/util/log"
	"github.com/cockroachdb/cockroach/util/stop"
)

var maxEpoch = flag.Int("maxEpoch", 10000, "Maximum epoch to simulate.")
var actionOutputFile = flag.String("action", "", "Output file that shows all actions taken in cluster.")
var epochOutputFile = flag.String("epoch", "", "Output file that shows stats for all epochs.")
var startingNodes = flag.Int("startingNodes", 3, "Number of initial nodes in the cluster.")
var scriptInputFile = flag.String("script", "default.script", "Input script file to describe simulated actions.")

func main() {
	stopper := stop.NewStopper()
	defer stopper.Stop()

	flag.Parse()
	// Give the flags some boundaries.
	if *startingNodes < 0 {
		*startingNodes = 0
	}
	if *maxEpoch < 0 {
		*maxEpoch = 0
	}

	// Clean the output file strings so they can be compared easily or set them
	// to nil if there is no file.
	if len(*actionOutputFile) > 0 {
		*actionOutputFile = filepath.Clean(*actionOutputFile)
	} else {
		actionOutputFile = nil
	}
	if len(*epochOutputFile) > 0 {
		*epochOutputFile = filepath.Clean(*epochOutputFile)
	} else {
		epochOutputFile = nil
	}
	*scriptInputFile = filepath.Clean(*scriptInputFile)

	fmt.Printf("A simulation of the cluster's rebalancing.\n\n")
	fmt.Printf("Maximum Epoch for simulation set to %d.\n", *maxEpoch)
	fmt.Printf("Cluster is starting with %d nodes.\n", *startingNodes)
	fmt.Printf("Script file is %s\n", *scriptInputFile)

	epochWriter := io.MultiWriter(os.Stdout)
	actionWriter := io.MultiWriter(os.Stdout)

	// Do we have an action output file?
	if actionOutputFile != nil && len(*actionOutputFile) > 0 {
		actionOutputF, err := os.OpenFile(*actionOutputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("Could not create or open action file:%s - %s", *actionOutputFile, err)
			return
		}
		defer actionOutputF.Close()
		fmt.Printf("Action Output will be written to %s.\n", *actionOutputFile)
		actionWriter = io.MultiWriter(os.Stdout, actionOutputF)
	} else {
		fmt.Printf("Action Output will only be written to console.\n")
	}

	// Do we have an epoch output file?
	if epochOutputFile != nil && len(*epochOutputFile) > 0 {
		// Is it the same as the action output file?
		if actionOutputFile != nil && *actionOutputFile == *epochOutputFile {
			epochWriter = actionWriter
			fmt.Printf("Epoch Output will also be written to %s.\n", *epochOutputFile)
		} else {
			epochOutputF, err := os.OpenFile(*epochOutputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				log.Fatalf("Could not create or open epoch file:%s - %s", *epochOutputFile, err)
				return
			}
			defer epochOutputF.Close()
			fmt.Printf("Epoch Output will be written to %s.\n", *epochOutputFile)
			epochWriter = io.MultiWriter(os.Stdout, epochOutputF)
		}
	} else {
		fmt.Printf("Epoch Output will only be written to console.\n")
	}

	fmt.Printf("\nParsing Script:\n")
	s, err := newScript(*maxEpoch, *scriptInputFile)
	if err != nil {
		log.Fatalf("Could not correctly parse script file:%s - %s", *scriptInputFile, err)
		return
	}

	fmt.Printf("\nPreparing Cluster:\n")
	c := createCluster(stopper, *startingNodes, epochWriter, actionWriter, s)

	// Run until stable or at the 100th epoch.
	fmt.Printf("\nRunning Simulation:\n")
	c.OutputEpochHeader()
	c.flush()
	for c.runEpoch() != true {
	}
	fmt.Println(c)
}

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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/cockroachdb/cockroach/util"
)

// Operation enumerates the possible scripted operations that can occur to a
// cluster.
type Operation int

// These are the possible scripted operations.
const (
	OpSplitRange Operation = iota
	OpAddNode
	OpExit
	// TODO(brma): add optoinal value to addnode to indicated size
	// TODO(bram): consider many other operations here.
	//	OpMergeRangeRandom
	//	OpMergeRangeFirst
	//  OpKillNodeRandom
	//  OpKillNodeFirst
	//  OpAddStoreRandom
	//  OpAddStore
	//	OpKillStoreRandom
	//	OpKillStoreFirst
)

var operations = [...]string{
	OpSplitRange: "splitrange",
	OpAddNode:    "addnode",
	OpExit:       "exit",
}

// String returns the string version of the Operation enumerable.
func (operation Operation) String() string {
	return operations[operation]
}

// operationFromString returns the operation corresponding with the passed in
// string.
func operationFromString(s string) (Operation, bool) {
	for i, op := range operations {
		if op == s {
			return Operation(i), true
		}
	}
	return 0, false
}

// OperationOption enumerates the possible options for a scripted operation.
type OperationOption int

// These are the possible operations options.
const (
	OpOpSet OperationOption = iota
	OpOpRandom
	OpOpFirst
	OpOpLast
)

var operationOptions = [...]string{
	OpOpSet:    "",
	OpOpRandom: "random",
	OpOpFirst:  "first",
	OpOpLast:   "last",
}

// String returns the string version of the OperationOption enumerable.
func (operationOption OperationOption) String() string {
	return operationOptions[operationOption]
}

// operationOptionFromString returns the option corresponding with the passed
// in string. If the value is opOpSet, then the value is parsed into the
// returned int.
func operationOptionFromString(s string) (OperationOption, int, error) {
	for i, opOption := range operationOptions {
		if opOption == s {
			return OperationOption(i), 0, nil
		}
	}
	value, err := strconv.Atoi(s)
	return OpOpSet, int(value), err
}

// Action is a single scripted action.
type Action struct {
	operation Operation
	option    OperationOption
	value     int
}

// ActionDetails contains an action and all of the metadata surrounding that
// action.
// First is the first epoch in which the action occurs.
// Every is the interval for how often the action re-occurs.
// Last is the final epoch in which the action can occur.
// Repeat is how many times an action occurs at each epoch in which it fires.
type ActionDetails struct {
	Action
	first  int
	every  int
	last   int
	repeat int
}

func (ad ActionDetails) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s", operations[ad.Action.operation])

	if ad.Action.option == OpOpSet {
		if ad.Action.value > 0 {
			fmt.Fprintf(&buf, "\tOption:%d", ad.Action.value)
		} else {
			fmt.Fprintf(&buf, "\t ")
		}
	} else {
		fmt.Fprintf(&buf, "\tOption:%s", operationOptions[ad.Action.option])
	}

	fmt.Fprintf(&buf, "\tFirst:%d", ad.first)
	if ad.last > 0 {
		fmt.Fprintf(&buf, "\tLast:%d", ad.last)
	} else {
		fmt.Fprintf(&buf, "\t")
	}
	if ad.every > 0 {
		fmt.Fprintf(&buf, "\tEvery:%d", ad.every)
	} else {
		fmt.Fprintf(&buf, "\t")
	}
	if ad.repeat > 1 {
		fmt.Fprintf(&buf, "\tRepeat:%d", ad.repeat)
	}
	return buf.String()
}

// Script contains a list of all the scripted actions for cluster simulation.
type Script struct {
	maxEpoch int
	actions  map[int][]Action
}

// createScript creates a new Script.
func createScript(maxEpoch int, scriptFile string) (Script, error) {
	s := Script{
		maxEpoch: maxEpoch,
		actions:  make(map[int][]Action),
	}

	return s, s.parse(scriptFile)
}

// addActionAtEpoch adds an action at a specific epoch to the s.actions map.
func (s *Script) addActionAtEpoch(action Action, repeat, epoch int) {
	for i := 0; i < repeat; i++ {
		s.actions[epoch] = append(s.actions[epoch], action)
	}
}

// addAction adds an action to the script.
func (s *Script) addAction(details ActionDetails) {
	// Always add the action at the first epoch. This way, single actions can
	// just set last and every to 0.
	last := details.last
	if last < details.first {
		last = details.first
	}

	// If every is less than 1, set it to maxEpoch so we never repeat, but do
	// exit the loop.
	every := details.every
	if every < 1 {
		every = s.maxEpoch
	}

	// Always set repeat to 1 if it is below zero.
	repeat := details.repeat
	if repeat < 1 {
		repeat = 1
	}

	// Save all the times this action should occur in s.actions.
	for currentEpoch := details.first; currentEpoch <= last && currentEpoch <= s.maxEpoch; currentEpoch += every {
		s.addActionAtEpoch(details.Action, repeat, currentEpoch)
	}
}

// getActions returns the list of actions for a specific epoch.
func (s *Script) getActions(epoch int) []Action {
	if epoch > s.maxEpoch {
		return []Action{{operation: OpExit}}
	}
	return s.actions[epoch]
}

// parse loads all the operations from a script file into the actions map. The
// format is described in the default.script file.
func (s *Script) parse(scriptFile string) error {
	file, err := os.Open(scriptFile)
	if err != nil {
		return err
	}
	defer file.Close()

	tw := tabwriter.NewWriter(os.Stdout, 12, 4, 2, ' ', 0)

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		line = strings.ToLower(line)
		line = strings.TrimSpace(line)

		// Ignore empty and commented out lines.
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Split the line by space.
		elements := strings.Split(line, " ")
		if len(elements) < 2 {
			return util.Errorf("line %d has too few elements: %s", lineNumber, line)
		}
		if len(elements) > 5 {
			return util.Errorf("line %d has too many elements: %s", lineNumber, line)
		}

		// Split the operation into operation-value.
		subElements := strings.Split(elements[0], "-")
		if len(subElements) > 2 {
			return util.Errorf("line %d operation has more than one value: %s", lineNumber, elements[0])
		}
		op, found := operationFromString(subElements[0])
		if !found {
			return util.Errorf("line %d operation could not be found: %s", lineNumber, subElements[0])
		}

		details := ActionDetails{Action: Action{operation: op}}
		// Do we have a value/number?
		if len(subElements) == 2 {
			if details.Action.option, details.Action.value, err = operationOptionFromString(subElements[1]); err != nil {
				return util.Errorf("line %d operation option could not be found or parsed: %s",
					lineNumber, subElements[1])
			}
		}

		// Get the first epoch.
		if details.first, err = strconv.Atoi(elements[1]); err != nil {
			return util.Errorf("line %d FIRST could not be parsed: %s", lineNumber, elements[1])
		}

		// Get the last epoch.
		if len(elements) > 2 {
			if details.last, err = strconv.Atoi(elements[2]); err != nil {
				return util.Errorf("line %d LAST could not be parsed: %s", lineNumber, elements[2])
			}
		}

		// Get the every value.
		if len(elements) > 3 {
			if details.every, err = strconv.Atoi(elements[3]); err != nil {
				return util.Errorf("line %d EVERY could not be parsed: %s", lineNumber, elements[3])
			}
		}

		// Get the repeat value.
		if len(elements) > 4 {
			if details.repeat, err = strconv.Atoi(elements[4]); err != nil {
				return util.Errorf("line %d REPEAT could not be parsed: %s", lineNumber, elements[4])
			}
		}

		s.addAction(details)
		fmt.Fprintf(tw, "%d:\t%s\n", lineNumber, details)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	_ = tw.Flush()

	return nil
}

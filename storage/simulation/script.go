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

// Operation enumerates the possible scripted operations that can occur to a
// cluster.
type Operation int

// These are the possible scripted operations.
const (
	OpSplitRange Operation = iota
	OpAddNode
	OpExit
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

// OperationValue enumerates the possible values for a scripted operation.
type OperationValue int

// These are the possible operations values.
const (
	OpValSet OperationValue = iota
	OpValRandom
	OpValFirst
	OpValLast
)

var operationValues = [...]string{
	OpValSet:    "",
	OpValRandom: "random",
	OpValFirst:  "first",
	OpValLast:   "last",
}

// String returns the string version of the OperationValue enumerable.
func (operationValue OperationValue) String() string {
	return operationValues[operationValue]
}

// Action is a single scripted action.
type Action struct {
	operation   Operation
	value       OperationValue
	valueNumber int
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

// Script contains a list of all the scripted actions for cluster simulation.
type Script struct {
	maxEpoch int
	actions  map[int][]Action
}

// newScript creates a new Script.
// TODO(bram): Add parsing logic.
func newScript(maxEpoch int) *Script {
	return &Script{
		maxEpoch: maxEpoch,
		actions:  make(map[int][]Action),
	}
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

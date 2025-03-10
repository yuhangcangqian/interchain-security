package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/google/go-cmp/cmp"

	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// an isolated test case for a proposal submission
var proposalSubmissionSteps = []Step{
	{SubmitTextProposalAction{Title: "Proposal 1", Description: "Description 1"}, State{}},
}

// an isolated test case for a state check involving a proposal
var proposalInStateSteps = []Step{
	{
		Action: SubmitConsumerRemovalProposalAction{},
		State: State{
			ChainID("provi"): ChainState{
				Proposals: &map[uint]Proposal{
					1: ConsumerRemovalProposal{
						Deposit: 10000001,
						Chain:   ChainID("foo"),
						Status:  gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD.String(),
					},
				},
			},
		},
	},
}

// Checks that writing, then parsing a trace results in the same trace.
func TestWriterThenParser(t *testing.T) {
	tests := map[string]struct {
		trace []Step
	}{
		"proposalSubmission":    {proposalSubmissionSteps},
		"proposalInState":       {proposalInStateSteps},
		"start_provider_chain":  {stepStartProviderChain()},
		"happyPath":             {happyPathSteps},
		"democracy":             {democracyUnregisteredDenomSteps},
		"slashThrottle":         {slashThrottleSteps},
		"multipleConsumers":     {multipleConsumers},
		"shorthappy":            {shortHappyPathSteps},
		"democracyRewardsSteps": {democracyRegisteredDenomSteps},
	}

	dir := t.TempDir() 

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(dir, "trace.json")
			err := WriteAndReadTrace(GlobalJSONParser, GlobalJSONWriter, tc.trace, filename)
			if err != nil {
				log.Fatalf("got error for testcase %v: %s", name, err)
			}
		})
	}
}

// Checks that writing a trace does not result in an error.
func TestWriteExamples(t *testing.T) {
	tests := map[string]struct {
		trace []Step
	}{
		"happyPath":             {happyPathSteps},
		"democracy":             {democracyUnregisteredDenomSteps},
		"slashThrottle":         {slashThrottleSteps},
		"multipleConsumers":     {multipleConsumers},
		"shorthappy":            {shortHappyPathSteps},
		"democracyRewardsSteps": {democracyRegisteredDenomSteps},
		"consumer-misbehaviour": {consumerMisbehaviourSteps},
		"consumer-double-sign":  {consumerDoubleSignSteps},
	}

	dir := "tracehandler_testdata"

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(dir, name+".json")
			err := GlobalJSONWriter.WriteTraceToFile(filename, tc.trace)
			if err != nil {
				t.Fatalf("error writing trace to file: %v", err)
			}
		})
	}
}

func TestMarshalAndUnmarshalChainState(t *testing.T) {
	tests := map[string]struct {
		chainState ChainState
	}{
		"consumer addition proposal": {ChainState{
			ValBalances: &map[ValidatorID]uint{
				ValidatorID("alice"): 9489999999,
				ValidatorID("bob"):   9500000000,
			},
			Proposals: &map[uint]Proposal{
				2: ConsumerAdditionProposal{
					Deposit:       10000001,
					Chain:         ChainID("test"),
					SpawnTime:     0,
					InitialHeight: clienttypes.Height{RevisionNumber: 5, RevisionHeight: 5},
					Status:        gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD.String(),
				},
			},
		}},
		"IBC transfer update params": {ChainState{
			ValBalances: &map[ValidatorID]uint{
				ValidatorID("alice"): 9889999998,
				ValidatorID("bob"):   9960000001,
			},
			Proposals: &map[uint]Proposal{
				1: IBCTransferParamsProposal{
					Deposit: 10000001,
					Status:  gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD.String(),
					Params:  IBCTransferParams{SendEnabled: true, ReceiveEnabled: true},
				},
			},
		}},
		"consumer removal proposal": {ChainState{
			Proposals: &map[uint]Proposal{
				5: ConsumerRemovalProposal{
					Deposit: 10000001,
					Chain:   ChainID("test123"),
					Status:  gov.ProposalStatus_PROPOSAL_STATUS_PASSED.String(),
				},
			},
			ValBalances: &map[ValidatorID]uint{
				ValidatorID("bob"): 9500000000,
			},
			ConsumerChains: &map[ChainID]bool{}, // Consumer chain is now removed
		}},
		"text-proposal": {ChainState{
			ValPowers: &map[ValidatorID]uint{
				ValidatorID("alice"): 509,
				ValidatorID("bob"):   500,
				ValidatorID("carol"): 495,
			},
			ValBalances: &map[ValidatorID]uint{
				ValidatorID("bob"): 9500000000,
			},
			Proposals: &map[uint]Proposal{
				// proposal does not exist
				10: TextProposal{},
			},
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := MarshalAndUnmarshalChainState(tc.chainState)
			if err != nil {
				t.Fatalf("MarshalAndUnmarshalChainState: %s", err.Error())
			}
		})
	}
}

func MarshalAndUnmarshalChainState(chainState ChainState) error {
	jsonobj, err := json.Marshal(chainState)
	if err != nil {
		return fmt.Errorf("error marshalling chain state: %v", err)
	}

	var got *ChainState
	err = json.Unmarshal(jsonobj, &got)
	if err != nil {
		return fmt.Errorf("error unmarshalling chain state: %v", err)
	}

	diff := cmp.Diff(chainState, *got)
	if diff != "" {
		log.Print(string(jsonobj))
		return fmt.Errorf("marshaled and unmarshaled ChainState don't match, diff=%s", diff)
	}

	return nil
}

func WriteAndReadTrace(parser TraceParser, writer TraceWriter, trace []Step, tmp_filepath string) error {
	err := writer.WriteTraceToFile(tmp_filepath, trace)
	if err != nil {
		return fmt.Errorf("error writing trace to file: %v", err)
	}

	got, err := GlobalJSONParser.ReadTraceFromFile(tmp_filepath)
	if err != nil {
		return fmt.Errorf("got error reading trace from file: %v", err)
	}
	diff := cmp.Diff(trace, got, cmp.AllowUnexported(Step{}))
	if diff != "" {
		return fmt.Errorf("Got a difference (-want +got):\n%s", diff)
	}
	return nil
}

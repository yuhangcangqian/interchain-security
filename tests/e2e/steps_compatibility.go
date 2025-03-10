package main

// Compatibility steps comprise a reduced set of actions suited to perform
// sanity checks across different ICS versions.

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	providertypes "github.com/cosmos/interchain-security/v7/x/ccv/provider/types"
)

func compstepStartProviderChain() []Step {
	return []Step{
		{
			Action: StartChainAction{
				Chain: ChainID("provi"),
				Validators: []StartChainValidator{
					{Id: ValidatorID("bob"), Stake: 500000000, Allocation: 10000000000},
					{Id: ValidatorID("alice"), Stake: 500000000, Allocation: 10000000000},
					{Id: ValidatorID("carol"), Stake: 500000000, Allocation: 10000000000},
				},
			},
			State: State{
				ChainID("provi"): ChainState{
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("alice"): 9500000000,
						ValidatorID("bob"):   9500000000,
						ValidatorID("carol"): 9500000000,
					},
				},
			},
		},
	}
}

func compstepsStartConsumerChain(consumerName string, proposalIndex, chainIndex uint, setupTransferChans bool) []Step {
	s := []Step{
		{
			Action: SubmitConsumerAdditionProposalAction{
				Chain:         ChainID("provi"),
				From:          ValidatorID("alice"),
				Deposit:       10000001,
				ConsumerChain: ChainID(consumerName),
				SpawnTime:     0,
				InitialHeight: clienttypes.Height{RevisionNumber: 0, RevisionHeight: 1},
				TopN:          100,
			},
			State: State{
				ChainID("provi"): ChainState{
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("alice"): 9489999999,
						ValidatorID("bob"):   9500000000,
					},
					Proposals: &map[uint]Proposal{
						proposalIndex: ConsumerAdditionProposal{
							Deposit:       10000001,
							Chain:         ChainID(consumerName),
							SpawnTime:     0,
							InitialHeight: clienttypes.Height{RevisionNumber: 0, RevisionHeight: 1},
							Status:        gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD.String(), // breaking change in SDK: gov.ProposalStatus(gov.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD).String(),
						},
					},
					// not supported across major versions
					// ProposedConsumerChains: &[]string{consumerName},
				},
			},
		},
		// add a consumer key before the chain starts
		// the key will be present in consumer genesis initial_val_set
		{
			Action: AssignConsumerPubKeyAction{
				Chain:          ChainID(consumerName),
				Validator:      ValidatorID("carol"),
				ConsumerPubkey: getDefaultValidators()[ValidatorID("carol")].ConsumerValPubKey,
				// consumer chain has not started
				// we don't need to reconfigure the node
				// since it will start with consumer key
				ReconfigureNode: false,
			},
			State: State{
				ChainID(consumerName): ChainState{
					AssignedKeys: &map[ValidatorID]string{
						ValidatorID("carol"): getDefaultValidators()[ValidatorID("carol")].ConsumerValconsAddressOnProvider,
					},
					ProviderKeys: &map[ValidatorID]string{
						ValidatorID("carol"): getDefaultValidators()[ValidatorID("carol")].ValconsAddress,
					},
				},
			},
		},
		{
			// op should fail - key already assigned by another validator
			Action: AssignConsumerPubKeyAction{
				Chain:     ChainID(consumerName),
				Validator: ValidatorID("bob"),
				// same pub key as carol
				ConsumerPubkey:  getDefaultValidators()[ValidatorID("carol")].ConsumerValPubKey,
				ReconfigureNode: false,
				ExpectError:     true,
				ExpectedError:   providertypes.ErrConsumerKeyInUse.Error(),
			},
			State: State{
				ChainID(consumerName): ChainState{
					AssignedKeys: &map[ValidatorID]string{
						ValidatorID("carol"): getDefaultValidators()[ValidatorID("carol")].ConsumerValconsAddressOnProvider,
						ValidatorID("bob"):   "",
					},
					ProviderKeys: &map[ValidatorID]string{
						ValidatorID("carol"): getDefaultValidators()[ValidatorID("carol")].ValconsAddress,
					},
				},
			},
		},
		{
			Action: VoteGovProposalAction{
				Chain:      ChainID("provi"),
				From:       []ValidatorID{ValidatorID("alice"), ValidatorID("bob"), ValidatorID("carol")},
				Vote:       []string{"yes", "yes", "yes"},
				PropNumber: proposalIndex,
			},
			State: State{
				ChainID("provi"): ChainState{
					Proposals: &map[uint]Proposal{
						proposalIndex: ConsumerAdditionProposal{
							Deposit:       10000001,
							Chain:         ChainID(consumerName),
							SpawnTime:     0,
							InitialHeight: clienttypes.Height{RevisionNumber: 0, RevisionHeight: 1},
							Status:        gov.ProposalStatus_PROPOSAL_STATUS_PASSED.String(),
						},
					},
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("alice"): 9500000000,
						ValidatorID("bob"):   9500000000,
					},
				},
			},
		},
		{
			Action: StartConsumerChainAction{
				ConsumerChain: ChainID(consumerName),
				ProviderChain: ChainID("provi"),
				Validators: []StartChainValidator{
					{Id: ValidatorID("bob"), Stake: 500000000, Allocation: 10000000000},
					{Id: ValidatorID("alice"), Stake: 500000000, Allocation: 10000000000},
					{Id: ValidatorID("carol"), Stake: 500000000, Allocation: 10000000000},
				},
			},
			State: State{
				ChainID("provi"): ChainState{
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("alice"): 9500000000,
						ValidatorID("bob"):   9500000000,
						ValidatorID("carol"): 9500000000,
					},
					// not supported
					// ProposedConsumerChains: &[]string{},
				},
				ChainID(consumerName): ChainState{
					ValBalances: &map[ValidatorID]uint{
						ValidatorID("alice"): 10000000000,
						ValidatorID("bob"):   10000000000,
						ValidatorID("carol"): 10000000000,
					},
				},
			},
		},
		{
			Action: AddIbcConnectionAction{
				ChainA:  ChainID(consumerName),
				ChainB:  ChainID("provi"),
				ClientA: 0,
				ClientB: chainIndex,
			},
			State: State{},
		},
		{
			Action: AddIbcChannelAction{
				ChainA:      ChainID(consumerName),
				ChainB:      ChainID("provi"),
				ConnectionA: 0,
				PortA:       "consumer", // TODO: check port mapping
				PortB:       "provider",
				Order:       "ordered",
			},
			State: State{},
		},
	}

	// currently only used in democracy tests
	if setupTransferChans {
		s = append(s, Step{
			Action: TransferChannelCompleteAction{
				ChainA:      ChainID(consumerName),
				ChainB:      ChainID("provi"),
				ConnectionA: 0,
				PortA:       "transfer",
				PortB:       "transfer",
				Order:       "unordered",
				ChannelA:    1,
				ChannelB:    1,
			},
			State: State{},
		})
	}
	return s
}

// starts provider and consumer chains specified in consumerNames
// setupTransferChans will establish a channel for fee transfers between consumer and provider
func compstepsStartChains(consumerNames []string, setupTransferChans bool) []Step {
	s := compstepStartProviderChain()
	for i, consumerName := range consumerNames {
		s = append(s, compstepsStartConsumerChain(consumerName, uint(i+1), uint(i), setupTransferChans)...)
	}

	return s
}

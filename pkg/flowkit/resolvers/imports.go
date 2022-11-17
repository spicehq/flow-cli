/*
 * Flow CLI
 *
 * Copyright 2019 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package resolvers

import (
	"fmt"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

// ImportResolver contains collection of Cadence programs and logic how to resolve imports.
//
// Containing functionality to build a dependency tree between programs and sort them based on that.
type ImportResolver struct {
	programs           []*Program
	loader             Loader
	aliases            map[string]string
	programsByLocation map[string]*Program
}

func NewImportResolver(loader Loader, aliases map[string]string) *ImportResolver {
	return &ImportResolver{
		loader:             loader,
		aliases:            aliases,
		programsByLocation: make(map[string]*Program),
	}
}

func (c *ImportResolver) Programs() []*Program {
	return c.programs
}

// Sort contracts by deployment order.
//
// Order of sorting is dependent on the possible imports contract contains, since
// any imported contract must be deployed before deploying the contract with that import.
// Only applicable to contracts.
func (c *ImportResolver) Sort() error {
	for _, p := range c.programs {
		if !p.isContract() {
			return fmt.Errorf("sorting is only possible for contracts")
		}
	}

	err := c.ResolveImports()
	if err != nil {
		return err
	}

	sorted, err := sortByDeploymentOrder(c.programs)
	if err != nil {
		return err
	}

	c.programs = sorted
	return nil
}

func (c *ImportResolver) Add(
	location string,
	accountAddress flow.Address,
	accountName string,
	args []cadence.Value,
) (*Program, error) {
	contractCode, err := c.loader.Load(location)
	if err != nil {
		return nil, err
	}

	contract, err := newProgram(
		len(c.programs),
		location,
		string(contractCode),
		accountAddress,
		accountName,
		args,
	)
	if err != nil {
		return nil, err
	}

	c.programs = append(c.programs, contract)
	c.programsByLocation[contract.location] = contract

	return contract, nil
}

// ResolveImports checks every program import and builds a dependency tree.
func (c *ImportResolver) ResolveImports() error {
	for _, program := range c.programs {
		for _, location := range program.imports() {
			importPath := location // TODO: c.loader.Normalize(program.source, source)
			importAlias, isAlias := c.aliases[importPath]
			importContract, isContract := c.programsByLocation[importPath]

			if isContract {
				program.addDependency(location, importContract)
			} else if isAlias {
				program.addAlias(location, flow.HexToAddress(importAlias))
			} else {
				return fmt.Errorf("import from %s could not be found: %s, make sure import path is correct", program.Name(), importPath)
			}
		}
	}

	return nil
}

// sortByDeploymentOrder sorts the given set of contracts in order of deployment.
//
// The resulting ordering ensures that each contract is deployed after all of its
// dependencies are deployed. This function returns an error if an import cycle exists.
//
// This function constructs a directed graph in which contracts are nodes and imports are edges.
// The ordering is computed by performing a topological sort on the constructed graph.
func sortByDeploymentOrder(contracts []*Program) ([]*Program, error) {
	g := simple.NewDirectedGraph()

	for _, c := range contracts {
		g.AddNode(c)
	}

	for _, c := range contracts {
		for _, dep := range c.dependencies {
			g.SetEdge(g.NewEdge(dep, c))
		}
	}

	sorted, err := topo.SortStabilized(g, nil)
	if err != nil {
		switch topoErr := err.(type) {
		case topo.Unorderable:
			return nil, &CyclicImportError{Cycles: nodeSetsToContractSets(topoErr)}
		default:
			return nil, err
		}
	}

	return nodesToContracts(sorted), nil
}

func nodeSetsToContractSets(nodes [][]graph.Node) [][]*Program {
	contracts := make([][]*Program, len(nodes))

	for i, s := range nodes {
		contracts[i] = nodesToContracts(s)
	}

	return contracts
}

func nodesToContracts(nodes []graph.Node) []*Program {
	contracts := make([]*Program, len(nodes))

	for i, s := range nodes {
		contracts[i] = s.(*Program)
	}

	return contracts
}

// CyclicImportError is returned when contract contain cyclic imports one to the
// other which is not possible to be resolved and deployed.
type CyclicImportError struct {
	Cycles [][]*Program
}

func (e *CyclicImportError) contractNames() [][]string {
	cycles := make([][]string, 0, len(e.Cycles))

	for _, cycle := range e.Cycles {
		contracts := make([]string, 0, len(cycle))
		for _, contract := range cycle {
			contracts = append(contracts, contract.Name())
		}

		cycles = append(cycles, contracts)
	}

	return cycles
}

func (e *CyclicImportError) Error() string {
	return fmt.Sprintf(
		"contracts: import cycle(s) detected: %v",
		e.contractNames(),
	)
}

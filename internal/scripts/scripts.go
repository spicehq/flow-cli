/*
 * Flow CLI
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package scripts

import (
	"bytes"
	"fmt"

	"github.com/onflow/cadence"
	"github.com/spf13/cobra"

	"github.com/onflow/flow-cli/pkg/flowcli/util"
)

var Cmd = &cobra.Command{
	Use:              "scripts",
	Short:            "Utilities to execute scripts",
	TraverseChildren: true,
}

func init() {
	ExecuteCommand.AddToParent(Cmd)
}

type ScriptResult struct {
	cadence.Value
}

// JSON convert result to JSON
func (r *ScriptResult) JSON() interface{} {
	result := make(map[string]interface{})
	result["result"] = r.Value.String()
	return result
}

// String convert result to string
func (r *ScriptResult) String() string {
	var b bytes.Buffer
	writer := util.CreateTabWriter(&b)

	_, _ = fmt.Fprintf(writer, "Result: %s\n", r.Value)

	writer.Flush()

	return b.String()
}

// Oneliner show result as one liner grep friendly
func (r *ScriptResult) Oneliner() string {
	return r.Value.String()
}
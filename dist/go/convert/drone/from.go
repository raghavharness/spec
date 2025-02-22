// Copyright 2022 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package drone

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	v2 "github.com/harness/yaml/dist/go"
	v1 "github.com/harness/yaml/dist/go/convert/drone/yaml"

	"github.com/ghodss/yaml"
)

// From converts the legacy drone yaml format to the
// unified yaml format.
func From(r io.Reader) ([]byte, error) {
	stages, err := v1.Parse(r)
	if err != nil {
		return nil, err
	}

	//
	// TODO perform Drone variable expansion
	//

	pipeline := new(v2.Pipeline)
	for _, from := range stages {
		if from == nil {
			continue
		}
		switch from.Kind {
		case v1.KindSecret: // TODO
		case v1.KindSignature: // TODO
		case v1.KindPipeline:
			pipeline.Stages = append(pipeline.Stages, &v2.Stage{
				Name: from.Name,
				Type: "ci",
				When: convertCond(from.Trigger),
				Spec: &v2.StageCI{
					Clone:    convertClone(from.Clone),
					Delegate: convertNode(from.Node),
					Envs:     copyenv(from.Environment),
					Platform: convertPlatform(from.Platform),
					Runtime:  convertRuntime(from),
					Steps:    convertSteps(from),

					// TODO support for stage.variables
					// TODO support for stage.tags
					// TODO support for stage.envs ?
					// TODO support for stage.volumes ?
					// TODO support for stage.pull_secrets ?
				},
			})
		}
	}

	return yaml.Marshal(pipeline)
}

// FromBytes converts the legacy drone yaml format to the
// unified yaml format.
func FromBytes(b []byte) ([]byte, error) {
	return From(
		bytes.NewBuffer(b),
	)
}

// FromString converts the legacy drone yaml format to the
// unified yaml format.
func FromString(s string) ([]byte, error) {
	return FromBytes(
		[]byte(s),
	)
}

// FromFile converts the legacy drone yaml format to the
// unified yaml format.
func FromFile(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return From(f)
}

func convertSteps(src *v1.Pipeline) []*v2.Step {
	var dst []*v2.Step
	for _, v := range src.Services {
		if v != nil {
			dst = append(dst, convertBackground(v))
		}
	}
	for _, v := range src.Steps {
		if v != nil {
			switch {
			case v.Detach:
				dst = append(dst, convertBackground(v))
			case isPlugin(v):
				dst = append(dst, convertPlugin(v))
			default:
				dst = append(dst, convertRun(v))
			}
		}
	}
	return dst
}

func convertPlugin(src *v1.Step) *v2.Step {
	return &v2.Step{
		Name: src.Name,
		Type: "plugin",
		When: convertCond(src.When),
		Spec: &v2.StepPlugin{
			Image:      src.Image,
			Privileged: src.Privileged,
			Pull:       convertPull(src.Pull),
			User:       src.User,
			Envs:       convertVariables(src.Environment),
			With:       convertSettings(src.Settings),
			Resources:  convertResourceLimits(&src.Resource),
			// Volumes       // FIX
		},
	}
}

func convertBackground(src *v1.Step) *v2.Step {
	return &v2.Step{
		Name: src.Name,
		Type: "background",
		When: convertCond(src.When),
		Spec: &v2.StepBackground{
			Image:      src.Image,
			Privileged: src.Privileged,
			Pull:       convertPull(src.Pull),
			Shell:      convertShell(src.Shell),
			User:       src.User,
			Entrypoint: convertEntrypoint(src.Entrypoint),
			Args:       convertArgs(src.Entrypoint, src.Command),
			Run:        convertScript(src.Commands),
			Envs:       convertVariables(src.Environment),
			Resources:  convertResourceLimits(&src.Resource),
			// Volumes       // FIX
		},
	}
}

func convertRun(src *v1.Step) *v2.Step {
	// TODO should harness support `dns`
	// TODO should harness support `dns_search`
	// TODO should harness support `extra_hosts`
	// TODO should harness support `network`
	// TODO should harness support `network_mode`
	// TODO should harness support `working_dir`
	return &v2.Step{
		Name: src.Name,
		Type: "script",
		When: convertCond(src.When),
		Spec: &v2.StepExec{
			Image:      src.Image,
			Privileged: src.Privileged,
			Pull:       convertPull(src.Pull),
			Shell:      convertShell(src.Shell),
			User:       src.User,
			Entrypoint: convertEntrypoint(src.Entrypoint),
			Args:       convertArgs(src.Entrypoint, src.Command),
			Run:        convertScript(src.Commands),
			Envs:       convertVariables(src.Environment),
			Resources:  convertResourceLimits(&src.Resource),
			// Volumes       // FIX
		},
	}
}

func convertResourceLimits(src *v1.Resources) *v2.Resources {
	if src.Limits.CPU == 0 && src.Limits.Memory == 0 {
		return nil
	}
	return &v2.Resources{
		Limits: &v2.Resource{
			Cpu:    v2.StringorInt(src.Requests.CPU),
			Memory: v2.MemStringorInt(src.Requests.Memory),
		},
	}
}

func convertResourceRequests(src *v1.Resources) *v2.Resources {
	if src.Requests.CPU == 0 && src.Requests.Memory == 0 {
		return nil
	}
	return &v2.Resources{
		Requests: &v2.Resource{
			Cpu:    v2.StringorInt(src.Requests.CPU),
			Memory: v2.MemStringorInt(src.Requests.Memory),
		},
	}
}

func convertEntrypoint(src []string) string {
	if len(src) == 0 {
		return ""
	} else {
		return src[0]
	}
}

func convertVariables(src map[string]*v1.Variable) map[string]string {
	dst := map[string]string{}
	for k, v := range src {
		switch {
		case v.Value != "":
			dst[k] = v.Value
		case v.Secret != "":
			dst[k] = fmt.Sprintf("${{ secrets.get(%q) }}", v.Secret) // TODO figure out secret syntax
		}
	}
	return dst
}

func convertSettings(src map[string]*v1.Parameter) map[string]interface{} {
	dst := map[string]interface{}{}
	for k, v := range src {
		switch {
		case v.Secret != "":
			dst[k] = fmt.Sprintf("${{ secrets.get(%q) }}", v.Secret)
		case v.Value != nil:
			dst[k] = v.Value
		}
	}
	return dst
}

func convertScript(src []string) string {
	if len(src) == 0 {
		return ""
	} else {
		return strings.Join(src, "\n")
	}
}

func convertArgs(src1, src2 []string) []string {
	if len(src1) == 0 {
		return src2
	} else {
		return append(src1[:1], src2...)
	}
}

func convertPull(src string) string {
	switch src {
	case "always":
		return "always"
	case "never":
		return "never"
	case "if-not-exists":
		return "if-not-exists"
	default:
		return ""
	}
}

func convertShell(src string) string {
	switch src {
	case "bash":
		return "bash"
	case "sh", "posix":
		return "sh"
	case "pwsh", "powershell":
		return "powershell"
	default:
		return ""
	}
}

func convertRuntime(src *v1.Pipeline) *v2.Runtime {
	if src.Type == "kubernetes" {
		return &v2.Runtime{
			Type: "kubernetes",
			Spec: &v2.RuntimeKube{
				// TODO should harness support `dns_config`
				// TODO should harness support `host_aliases`
				// TODO support for `tolerations`
				Annotations:    src.Metadata.Annotations,
				Labels:         src.Metadata.Labels,
				Namespace:      src.Metadata.Namespace,
				NodeSelector:   src.NodeSelector,
				Node:           src.NodeName,
				ServiceAccount: src.ServiceAccount,
				Resources:      convertResourceRequests(&src.Resource),
			},
		}
	}
	return &v2.Runtime{
		Type: "machine",
		Spec: v2.RuntimeMachine{},
	}
}

func convertClone(src v1.Clone) *v2.Clone {
	dst := new(v2.Clone)
	if v := src.Depth; v != 0 {
		dst.Depth = int64(v)
	}
	if v := src.Disable; v {
		dst.Disabled = true
	}
	if v := src.SkipVerify; v {
		dst.Insecure = true
	}
	if v := src.Trace; v {
		dst.Trace = true
	}
	return dst
}

func convertNode(src map[string]string) *v2.Delegate {
	if len(src) == 0 {
		return nil
	}
	dst := new(v2.Delegate)
	for k, v := range src {
		dst.Selectors = append(
			dst.Selectors, k+":"+v)
	}
	return dst
}

func convertPlatform(src v1.Platform) *v2.Platform {
	if src.Arch == "" && src.OS == "" {
		return nil
	}
	dst := new(v2.Platform)
	switch src.OS {
	case "windows":
		dst.Os = v2.OSWindows
	case "darwin":
		dst.Os = v2.OSDarwin
	default:
		dst.Os = v2.OSLinux
	}
	switch src.Arch {
	case "arm":
		dst.Arch = v2.ArchArm64
	case "arm64":
		dst.Arch = v2.ArchArm64
	default:
		dst.Arch = v2.ArchAmd64
	}
	return dst
}

func convertCond(src v1.Conditions) *v2.When {
	if isCondsEmpty(src) {
		return nil
	}

	exprs := map[string]*v2.Expr{}
	if expr := convertExpr(src.Action); expr != nil {
		exprs["action"] = expr
	}
	if expr := convertExpr(src.Branch); expr != nil {
		exprs["branch"] = expr
	}
	if expr := convertExpr(src.Cron); expr != nil {
		exprs["cron"] = expr
	}
	if expr := convertExpr(src.Event); expr != nil {
		exprs["event"] = expr
	}
	if expr := convertExpr(src.Instance); expr != nil {
		exprs["instance"] = expr
	}
	if expr := convertExpr(src.Paths); expr != nil {
		exprs["paths"] = expr
	}
	if expr := convertExpr(src.Ref); expr != nil {
		exprs["ref"] = expr
	}
	if expr := convertExpr(src.Repo); expr != nil {
		exprs["repo"] = expr
	}
	if expr := convertExpr(src.Status); expr != nil {
		exprs["status"] = expr
	}
	if expr := convertExpr(src.Target); expr != nil {
		exprs["target"] = expr
	}

	dst := new(v2.When)
	dst.Cond = []map[string]*v2.Expr{exprs}
	return dst
}

func convertExpr(src v1.Condition) *v2.Expr {
	if len(src.Include) != 0 {
		return &v2.Expr{In: src.Include}
	}
	if len(src.Exclude) != 0 {
		return &v2.Expr{
			Not: &v2.Expr{In: src.Include},
		}
	}
	return nil
}

func isCondsEmpty(src v1.Conditions) bool {
	return isCondEmpty(src.Action) &&
		isCondEmpty(src.Action) &&
		isCondEmpty(src.Branch) &&
		isCondEmpty(src.Cron) &&
		isCondEmpty(src.Event) &&
		isCondEmpty(src.Instance) &&
		isCondEmpty(src.Paths) &&
		isCondEmpty(src.Ref) &&
		isCondEmpty(src.Repo) &&
		isCondEmpty(src.Status) &&
		isCondEmpty(src.Target)
}

func isCondEmpty(src v1.Condition) bool {
	return len(src.Exclude) == 0 && len(src.Include) == 0
}

func isPlugin(src *v1.Step) bool {
	return len(src.Settings) > 0
}

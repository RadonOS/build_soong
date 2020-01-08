// Copyright 2019 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package java

import (
	"android/soong/android"
)

func init() {
	android.RegisterSingletonType("platform_compat_config_singleton", platformCompatConfigSingletonFactory)
	android.RegisterModuleType("platform_compat_config", platformCompatConfigFactory)
}

type platformCompatConfigSingleton struct {
	metadata android.Path
}

type platformCompatConfigProperties struct {
	Src *string `android:"path"`
}

type platformCompatConfig struct {
	android.ModuleBase

	properties     platformCompatConfigProperties
	installDirPath android.InstallPath
	configFile     android.OutputPath
	metadataFile   android.OutputPath
}

func (p *platformCompatConfig) compatConfigMetadata() android.OutputPath {
	return p.metadataFile
}

type platformCompatConfigIntf interface {
	compatConfigMetadata() android.OutputPath
}

var _ platformCompatConfigIntf = (*platformCompatConfig)(nil)

// compat singleton rules
func (p *platformCompatConfigSingleton) GenerateBuildActions(ctx android.SingletonContext) {

	var compatConfigMetadata android.Paths

	ctx.VisitAllModules(func(module android.Module) {
		if c, ok := module.(platformCompatConfigIntf); ok {
			metadata := c.compatConfigMetadata()
			compatConfigMetadata = append(compatConfigMetadata, metadata)
		}
	})

	if compatConfigMetadata == nil {
		// nothing to do.
		return
	}

	rule := android.NewRuleBuilder()
	outputPath := android.PathForOutput(ctx, "compat_config", "merged_compat_config.xml")

	rule.Command().
		BuiltTool(ctx, "process-compat-config").
		FlagForEachInput("--xml ", compatConfigMetadata).
		FlagWithOutput("--merged-config ", outputPath)

	rule.Build(pctx, ctx, "merged-compat-config", "Merge compat config")

	p.metadata = outputPath
}

func (p *platformCompatConfigSingleton) MakeVars(ctx android.MakeVarsContext) {
	if p.metadata != nil {
		ctx.Strict("INTERNAL_PLATFORM_MERGED_COMPAT_CONFIG", p.metadata.String())
	}
}

func (p *platformCompatConfig) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	rule := android.NewRuleBuilder()

	configFileName := p.Name() + ".xml"
	metadataFileName := p.Name() + "_meta.xml"
	p.configFile = android.PathForModuleOut(ctx, configFileName).OutputPath
	p.metadataFile = android.PathForModuleOut(ctx, metadataFileName).OutputPath
	path := android.PathForModuleSrc(ctx, String(p.properties.Src))

	rule.Command().
		BuiltTool(ctx, "process-compat-config").
		FlagWithInput("--jar ", path).
		FlagWithOutput("--device-config ", p.configFile).
		FlagWithOutput("--merged-config ", p.metadataFile)

	p.installDirPath = android.PathForModuleInstall(ctx, "etc", "compatconfig")
	rule.Build(pctx, ctx, configFileName, "Extract compat/compat_config.xml and install it")

}

func (p *platformCompatConfig) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "ETC",
		OutputFile: android.OptionalPathForPath(p.configFile),
		Include:    "$(BUILD_PREBUILT)",
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", p.installDirPath.ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", p.configFile.Base())
			},
		},
	}}
}

func platformCompatConfigSingletonFactory() android.Singleton {
	return &platformCompatConfigSingleton{}
}

func platformCompatConfigFactory() android.Module {
	module := &platformCompatConfig{}
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibFirst)
	return module
}

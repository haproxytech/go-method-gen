package common

type FuncRef struct {
	Pkg  string `yaml:"pkg"`
	Name string `yaml:"name"`
}

type OverrideFuncs struct {
	Equal *FuncRef `yaml:"equal"`
	Diff  *FuncRef `yaml:"diff"`
	Merge *FuncRef `yaml:"merge"`
}

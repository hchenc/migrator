package cmd

type MigratorFlages struct {
	Migration  migration  `yaml:"migration"`
	Datasource datasource `yaml:"datasource"`
}

type migration struct {
	Location string `yaml:"location"`
	Table    string `yaml:"table"`
}

type datasource struct {
	Password string `yaml:"password"`
	Url      string `yaml:"url"`
	User     string `yaml:"user"`
}

func (flags *MigratorFlages) convert(config, commandline *MigratorFlages) {

}



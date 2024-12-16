package configurer

type setupFn func(configurer Configurer) error

func baseSetup(configurer Configurer) error {
	return configurer.RunValidators()
}

func withIBC(setupHandler setupFn) setupFn {
	return func(configurer Configurer) error {
		if err := setupHandler(configurer); err != nil {
			return err
		}

		return configurer.RunIBC()
	}
}

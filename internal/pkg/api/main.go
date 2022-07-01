package api

type NucleusApi interface {
}

type NucleusService struct {
	Name                 string
	Runtime              string // nodejs | python | go
	IsPrivateService     bool
	EnvironmentVariables map[string]string
	Secrets              map[string]string
}

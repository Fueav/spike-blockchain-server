package model

type PinataParams struct {
	PinataOptions  PinataOptions  `json:"pinataOptions"`
	PinataMetaData PinataMetaData `json:"pinataMetadata"`
	PinataContent  string         `json:"pinataContent"`
}

type PinataOptions struct {
	CidVersion        int             `json:"cidVersion"`
	WrapWithDirectory bool            `json:"wrapWithDirectory"`
	CustomPinPolicy   CustomPinPolicy `json:"customPinPolicy"`
}

type PinataMetaData struct {
	Name      string            `json:"name"`
	Keyvalues map[string]string `json:"keyvalues"`
}

type Region struct {
	ID                      string `json:"id"`
	DesiredReplicationCount int    `json:"desiredReplicationCount"`
}

type CustomPinPolicy struct {
	Regions []Region `json:"regions"`
}

var (
	DefaultPinataConfig = PinataParams{
		PinataOptions: PinataOptions{
			CidVersion: 0,
			CustomPinPolicy: CustomPinPolicy{
				Regions: []Region{
					{
						ID:                      "FRA1",
						DesiredReplicationCount: 2,
					},
				},
			},
		},
		PinataMetaData: PinataMetaData{
			Name: "Default Pinata MetaData",
		},
	}
)

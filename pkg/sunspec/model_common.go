package sunspec

const (
	SUNS_MODEL_LENGTH_COMMON = 65 * 2 // 65 registers
)

type CommonModelData struct {
	modelData
}

func ReadModelCommon(model ReadableModel) (*CommonModelData, error) {
	if model.id != SUNS_MODEL_ID_COMMON {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if len(data.data) != SUNS_MODEL_LENGTH_COMMON {
		return nil, ErrInvalidModelLength
	}
	return &CommonModelData{modelData: data}, nil
}

func (data *CommonModelData) Manufacturer() string {
	return data.readString(0, 32)
}

func (data *CommonModelData) Model() string {
	return data.readString(16, 32)
}

func (data *CommonModelData) Version() string {
	return data.readString(40, 16)
}

func (data *CommonModelData) Serial() string {
	return data.readString(48, 32)
}

package sunspec

const (
	SUNS_MODEL_LENGTH_STATUS = 44 * 2 // 44 registers
)

type ConnectionState uint16

func (cs ConnectionState) Connected() bool {
	return cs&0x0001 == 1
}

func (cs ConnectionState) Available() bool {
	return cs&0x0002 == 1
}

func (cs ConnectionState) Operating() bool {
	return cs&0x0004 == 1
}

type StatusModelData struct {
	modelData
}

func ReadModelStatus(model ReadableModel) (*StatusModelData, error) {
	if model.id != SUNS_MODEL_ID_STATUS {
		return nil, ErrInvalidModelId
	}
	data, err := model.Read()
	if err != nil {
		return nil, err
	}
	if len(data.data) != SUNS_MODEL_LENGTH_STATUS {
		return nil, ErrInvalidModelLength
	}
	return &StatusModelData{modelData: data}, nil
}

func (data *StatusModelData) PVConnection() ConnectionState {
	return ConnectionState(data.readRegister(0))
}

func (data *StatusModelData) StorageConnection() ConnectionState {
	return ConnectionState(data.readRegister(1))
}

func (data *StatusModelData) ECPConnection() ConnectionState {
	return ConnectionState(data.readRegister(2))
}

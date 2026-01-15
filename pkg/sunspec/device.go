package sunspec

type SunspecDeviceModels struct {
	models []ModelHeader
}

func NewDeviceModels(models []ModelHeader) *SunspecDeviceModels {
	return &SunspecDeviceModels{
		models: models,
	}
}

func (device *SunspecDeviceModels) FindModelById(id uint16) *ModelHeader {
	for _, model := range device.models {
		if model.id == id {
			return &model
		}
	}
	return nil
}

func (device *SunspecDeviceModels) FindModelByIdRange(minId uint16, maxId uint16) *ModelHeader {
	for _, model := range device.models {
		if model.id >= minId && model.id <= maxId {
			return &model
		}
	}
	return nil
}

func (device *SunspecDeviceModels) CountModelsById(id uint16) int {

	count := 0
	for _, model := range device.models {
		if model.id == id {
			count++
		}
	}
	return count
}

func (device *SunspecDeviceModels) FindModelsById(id uint16) []ModelHeader {

	result := []ModelHeader{}
	for _, model := range device.models {
		if model.id == id {
			result = append(result, model)
		}
	}
	return result
}

func (device *SunspecDeviceModels) FindModelByIdAndIndex(id uint16, index int) *ModelHeader {

	currentIndex := 0
	for _, model := range device.models {
		if model.id == id {
			if currentIndex == index {
				return &model
			}
			currentIndex++
		}
	}
	return nil
}

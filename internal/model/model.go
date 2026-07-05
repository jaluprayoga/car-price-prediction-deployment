package model

import (
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// InitONNX initializes the ONNX Runtime with the shared library path.
func InitONNX(libPath string) error {
	ort.SetSharedLibraryPath(libPath)
	return ort.InitializeEnvironment()
}

// CleanupONNX destroys the ONNX Runtime environment.
func CleanupONNX() {
	ort.DestroyEnvironment()
}

// CarFeatures represents the preprocessed inputs for our ONNX model.
type CarFeatures struct {
	KmDriven     float32
	Age          float32
	Mileage      float32
	Engine       float32
	MaxPower     float32
	Seats        float32
	Fuel         string
	SellerType   string
	Transmission string
	Owner        string
}

// Predictor loads and executes inference on the ONNX model pipeline.
type Predictor struct {
	session      *ort.AdvancedSession
	inputs       []ort.Value
	inputTensors map[string]ort.Value
	outputTensor *ort.Tensor[float32]
	mu           sync.Mutex
}

// NewPredictor loads the model at modelPath and initializes input/output tensors.
func NewPredictor(modelPath string) (*Predictor, error) {
	inputNames := []string{
		"km_driven", "Age", "mileage", "engine", "max_power", "seats",
		"fuel", "seller_type", "transmission", "owner",
	}
	outputNames := []string{"variable"}

	shape := ort.NewShape(1, 1)
	inputs := make([]ort.Value, len(inputNames))
	inputTensors := make(map[string]ort.Value)

	// Numeric inputs (first 6)
	for i := 0; i < 6; i++ {
		tensor, err := ort.NewTensor(shape, []float32{0.0})
		if err != nil {
			// Cleanup previously allocated tensors on failure
			for j := 0; j < i; j++ {
				inputs[j].Destroy()
			}
			return nil, err
		}
		inputs[i] = tensor
		inputTensors[inputNames[i]] = tensor
	}

	// Categorical inputs (next 4)
	for i := 6; i < 10; i++ {
		tensor, err := ort.NewStringTensor(shape)
		if err != nil {
			// Cleanup previously allocated tensors on failure
			for j := 0; j < i; j++ {
				inputs[j].Destroy()
			}
			return nil, err
		}
		err = tensor.SetContents([]string{""})
		if err != nil {
			tensor.Destroy()
			for j := 0; j < i; j++ {
				inputs[j].Destroy()
			}
			return nil, err
		}
		inputs[i] = tensor
		inputTensors[inputNames[i]] = tensor
	}

	// Output tensor
	outputShape := ort.NewShape(1, 1)
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		for j := 0; j < len(inputs); j++ {
			inputs[j].Destroy()
		}
		return nil, err
	}

	session, err := ort.NewAdvancedSession(
		modelPath,
		inputNames,
		outputNames,
		inputs,
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		outputTensor.Destroy()
		for j := 0; j < len(inputs); j++ {
			inputs[j].Destroy()
		}
		return nil, err
	}

	return &Predictor{
		session:      session,
		inputs:       inputs,
		inputTensors: inputTensors,
		outputTensor: outputTensor,
	}, nil
}

// Destroy cleans up the session and tensors allocated in C memory.
func (p *Predictor) Destroy() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.session != nil {
		p.session.Destroy()
	}
	for _, tensor := range p.inputs {
		if tensor != nil {
			tensor.Destroy()
		}
	}
	if p.outputTensor != nil {
		p.outputTensor.Destroy()
	}
}

// Predict performs inference using the ONNX model, updating tensors thread-safely.
func (p *Predictor) Predict(features CarFeatures) (float32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Update numeric inputs
	p.inputTensors["km_driven"].(*ort.Tensor[float32]).GetData()[0] = features.KmDriven
	p.inputTensors["Age"].(*ort.Tensor[float32]).GetData()[0] = features.Age
	p.inputTensors["mileage"].(*ort.Tensor[float32]).GetData()[0] = features.Mileage
	p.inputTensors["engine"].(*ort.Tensor[float32]).GetData()[0] = features.Engine
	p.inputTensors["max_power"].(*ort.Tensor[float32]).GetData()[0] = features.MaxPower
	p.inputTensors["seats"].(*ort.Tensor[float32]).GetData()[0] = features.Seats

	// Update categorical inputs
	_ = p.inputTensors["fuel"].(*ort.StringTensor).SetContents([]string{features.Fuel})
	_ = p.inputTensors["seller_type"].(*ort.StringTensor).SetContents([]string{features.SellerType})
	_ = p.inputTensors["transmission"].(*ort.StringTensor).SetContents([]string{features.Transmission})
	_ = p.inputTensors["owner"].(*ort.StringTensor).SetContents([]string{features.Owner})

	// Run session
	err := p.session.Run()
	if err != nil {
		return 0, err
	}

	// Retrieve value
	predPrice := p.outputTensor.GetData()[0]
	return predPrice, nil
}

package gan_go

import (
	"fmt"

	"github.com/pkg/errors"
	"gorgonia.org/gorgonia"
)

// Generator Abstraction for discriminator part of GAN. It's simple neural network actually.
//
// Layers - simple sequence of layers
// out - alias to activated output of last layer
//
type Discriminator struct {
	Layers []*Layer
	out    *gorgonia.Node
}

// Out Returns reference to output node
func (net *Discriminator) Out() *gorgonia.Node {
	return net.out
}

// Learnables Returns learnables nodes
func (net *Discriminator) Learnables() gorgonia.Nodes {
	learnables := make(gorgonia.Nodes, 0, 2*len(net.Layers))
	for _, l := range net.Layers {
		if l != nil {
			if l.WeightNode != nil {
				learnables = append(learnables, l.WeightNode)
			}
			if l.BiasNode != nil {
				learnables = append(learnables, l.BiasNode)
			}
		}
	}
	return learnables
}

// Fwd Initializates feedforward for provided input
//
// input - Input node
// batchSize - batch size. If it's >= 2 then broadcast function will be applied
//
func (net *Discriminator) Fwd(input *gorgonia.Node, batchSize int) error {
	if len(net.Layers) == 0 {
		return fmt.Errorf("Discriminator must have one layer atleast")
	}
	if net.Layers[0] == nil {
		return fmt.Errorf("Discriminator's layer #0 is nil")
	}
	if net.Layers[0].WeightNode == nil {
		return fmt.Errorf("Discriminator's layer #0 WeightNode is nil")
	}
	tOp, err := gorgonia.Transpose(net.Layers[0].WeightNode)
	if err != nil {
		return errors.Wrap(err, "Can't transpose weights of Discriminator's layer #0")
	}
	firstLayerNonActivated := &gorgonia.Node{}
	switch net.Layers[0].Type {
	case LayerLinear:
		firstLayerNonActivated, err = gorgonia.Mul(input, tOp)
		if err != nil {
			return errors.Wrap(err, "Can't multiply input and weights of Discriminator's layer #0")
		}
		break
	default:
		return fmt.Errorf("Layer #0's type '%d' (uint16) is not handled [Discriminator]", net.Layers[0].Type)
	}

	gorgonia.WithName("discriminator_0")(firstLayerNonActivated)
	if net.Layers[0].BiasNode != nil {
		if batchSize < 2 {
			firstLayerNonActivated, err = gorgonia.Add(firstLayerNonActivated, net.Layers[0].BiasNode)
			if err != nil {
				return errors.Wrap(err, "Can't add bias to non-activated output of Discriminator's layer #0")
			}
		} else {
			firstLayerNonActivated, err = gorgonia.BroadcastAdd(firstLayerNonActivated, net.Layers[0].BiasNode, nil, []byte{0})
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Can't add [in broadcast term with batch_size = %d] bias to non-activated output of Discriminator's layer #0", batchSize))
			}
		}
	}
	firstLayerActivated, err := net.Layers[0].Activation(firstLayerNonActivated)
	if err != nil {
		return errors.Wrap(err, "Can't apply activation function to non-activated output of Discriminator's layer #0")
	}
	gorgonia.WithName("discriminator_activated_0")(firstLayerActivated)
	lastActivatedLayer := firstLayerActivated
	for i := 1; i < len(net.Layers); i++ {
		if net.Layers[i] == nil {
			return fmt.Errorf("Discriminator's layer #%d is nil", i)
		}
		if net.Layers[i].WeightNode == nil {
			return fmt.Errorf("Discriminator's layer's #%d WeightNode is nil", i)
		}
		tOp, err := gorgonia.Transpose(net.Layers[i].WeightNode)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Can't transpose weights of Discriminator's layer #%d", i))
		}

		layerNonActivated := &gorgonia.Node{}
		switch net.Layers[i].Type {
		case LayerLinear:
			layerNonActivated, err = gorgonia.Mul(lastActivatedLayer, tOp)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Can't multiply input and weights of Discriminator's layer #%d", i))
			}
			break
		default:
			return fmt.Errorf("Layer #%d's type '%d' (uint16) is not handled [Discriminator]", i, net.Layers[i].Type)
		}

		gorgonia.WithName(fmt.Sprintf("discriminator_%d", i))(layerNonActivated)
		if net.Layers[i].BiasNode != nil {
			if batchSize < 2 {
				layerNonActivated, err = gorgonia.Add(layerNonActivated, net.Layers[i].BiasNode)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Can't add bias to non-activated output of Discriminator's layer #%d", i))
				}
			} else {
				layerNonActivated, err = gorgonia.BroadcastAdd(layerNonActivated, net.Layers[i].BiasNode, nil, []byte{0})
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Can't add bias [in broadcast term with batch_size = %d] to non-activated output of Discriminator's layer #%d", batchSize, i))
				}
			}
		}
		layerActivated, err := net.Layers[i].Activation(layerNonActivated)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Can't apply activation function to non-activated output of Discriminator's layer #%d", i))
		}
		gorgonia.WithName(fmt.Sprintf("discriminator_activated_%d", i))(layerActivated)
		lastActivatedLayer = layerActivated
		if i == len(net.Layers)-1 {
			net.out = layerActivated
		}
	}
	return nil
}

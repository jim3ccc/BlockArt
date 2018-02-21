package blockartlib

import (
	"../HTML"
	"../shared"
	"net/rpc"
	"strconv"
)

type CanvasImpl struct {
	miner      *rpc.Client
	serverAddr string
	settings   shared.CanvasSettings
}

func (c *CanvasImpl) Call(serviceMethod string, args interface{}, reply interface{}) error {
	err := c.miner.Call(serviceMethod, args, reply)
	if err != nil {
		return err
	}
	return err
}

// Adds a new shape to the canvas.
// Can return the following errors:
// - DisconnectedError
// - InsufficientInkError
// - InvalidShapeSvgStringError
// - ShapeSvgStringTooLongError
// - ShapeOverlapError
// - OutOfBoundsError
func (c *CanvasImpl) AddShape(validateNum uint8, shapeType shared.ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error) {
	args := &shared.AddShapeArgs{ValidateNum: validateNum, ShapeType: shapeType, ShapeSvgString: shapeSvgString, Fill: fill, Stroke: stroke}
	reply := new(shared.AddShapeReply)
	err = c.Call("Miner.AddShape", args, reply)
	if err != nil {
		switch reply.ShapeHash {
		case "0":
			v, _ := strconv.Atoi(reply.BlockHash)
			err =  InsufficientInkError(v)
		case "1":
			err =  InvalidShapeSvgStringError("<path d=\"" + shapeSvgString + "\" stroke=\"" + stroke + "\" fill=" + fill + "\"/>")
		case "2":
			err =  ShapeSvgStringTooLongError("<path d=\"" + shapeSvgString + "\" stroke=\"" + stroke + "\" fill=" + fill + "\"/>")
		case "3":
			err =  ShapeOverlapError("<path d=\"" + shapeSvgString + "\" stroke=\"" + stroke + "\" fill=" + fill + "\"/>")
		case "4":
			err =  OutOfBoundsError{}
		default:
				err = DisconnectedError(c.serverAddr)
				return
		}
		reply.ShapeHash = ""
		return
	}
	return reply.ShapeHash, reply.BlockHash, reply.InkRemaining, nil
}

// Returns the encoding of the shape as an svg string.
// Can return the following errors:
// - DisconnectedError
// - InvalidShapeHashError
func (c *CanvasImpl) GetSvgString(shapeHash string) (svgString string, err error) {
	err = c.Call("Miner.GetSvgShape", shapeHash, &svgString)
	if err != nil {
		if svgString == "0" {
			err = InvalidShapeHashError(shapeHash)
		} else {
			err =  DisconnectedError(c.serverAddr)
		}
		svgString = ""
		return
	}
	return
}

// Returns the amount of ink currently available.
// Can return the following errors:
// - DisconnectedError
func (c *CanvasImpl) GetInk() (inkRemaining uint32, err error) {
	err = c.Call("Miner.GetInk", new(interface{}), &inkRemaining)
	if err != nil {
		return
	}
	return
}

// Removes a shape from the canvas.
// Can return the following errors:
// - DisconnectedError
// - ShapeOwnerError
func (c *CanvasImpl) DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error) {
	args := &shared.DeleteShapeArgs{ValidateNum: validateNum, ShapeHash: shapeHash}
	err = c.Call("Miner.DeleteShape", args, &inkRemaining)
	if err != nil {
		if inkRemaining == uint32(0) {
			err = ShapeOwnerError(shapeHash)
		} else {
			err = DisconnectedError(c.serverAddr)
		}
		return
	}
	return
}

// Retrieves hashes contained by a specific block.
// Can return the following errors:
// - DisconnectedError
// - InvalidBlockHashError
func (c *CanvasImpl) GetShapes(blockHash string) (shapeHashes []string, err error) {
	err = c.Call("Miner.GetShapes", blockHash, &shapeHashes)
	if err != nil {
		if shapeHashes[0] == "0" {
			err = InvalidBlockHashError(blockHash)
		} else {
			err= DisconnectedError(c.serverAddr)
		}
		shapeHashes  = append(shapeHashes[:0], shapeHashes[:1]...)
		return
	}
	return
}

// Returns the block hash of the genesis block.
// Can return the following errors:
// - DisconnectedError
func (c *CanvasImpl) GetGenesisBlock() (blockHash string, err error) {
	err = c.Call("Miner.GetGenesisBlock", new(interface{}), &blockHash)
	if err != nil {
		err = DisconnectedError(c.serverAddr)
		return
	}
	return
}

// Retrieves the children blocks of the block identified by blockHash.
// Can return the following errors:
// - DisconnectedError
// - InvalidBlockHashError
func (c *CanvasImpl) GetChildren(blockHash string) (blockHashes []string, err error) {
	err = c.Call("Miner.GetChildren", blockHash, &blockHashes)
	if err != nil {
		if blockHashes[0] == "0" {
			err =  InvalidBlockHashError(blockHash)
		} else {
			err =  DisconnectedError(c.serverAddr)
		}
		blockHashes  = append(blockHashes[:0], blockHashes[:1]...)
		return
	}
	return
}

func (c *CanvasImpl) GetCanvas() (err error) {
	svgArrays := []map[string]string{}
	err = c.Call("Miner.MakeSVGCanvas", new(interface{}), &svgArrays)
	if err != nil {
		return DisconnectedError(c.serverAddr)
	}
	for i, canvas := range svgArrays {
		index := strconv.Itoa(i)
		array := []string{}
		count := 0
		for _, v := range canvas {
			array = append(array, v)
			count++
		}
		HTML.CreateHtml(array, c.settings.CanvasXMax, c.settings.CanvasYMax, "Canvas"+index+".html")
	}
	return
}

// Closes the canvas/connection to the BlockArt network.
// - DisconnectedError
func (c *CanvasImpl) CloseCanvas() (inkRemaining uint32, err error) {
	err = c.Call("Miner.CloseCanvas", new(interface{}), &inkRemaining)
	if err != nil {
		err = DisconnectedError(c.serverAddr)
		return
	}
	c.miner = nil
	c.serverAddr = ""
	return
}

// Copyright (c) 2012 Google, Inc. All rights reserved.

package gopacket

import (
	"encoding/binary"
	"fmt"
)

type RUDP struct {
	SYN, ACK, EACK, RST, NUL bool
	Version                  uint8
	HeaderLength             uint8
	SrcPort, DstPort         uint8
	DataLength               uint16
	Seq, Ack, Checksum       uint32
	VariableHeaderArea       []byte
	// RUDPSyn contains SYN information for the RUDP packet,
	// if the SYN flag is set
	*RUDPHeaderSYN
	// RUDPEack contains EACK information for the RUDP packet,
	// if the EACK flag is set.
	*RUDPHeaderEACK
}

type RUDPHeaderSYN struct {
	MaxOutstandingSegments, MaxSegmentSize, OptionFlags uint16
}

type RUDPHeaderEACK struct {
	SeqsReceivedOK []uint32
}

// LayerType returns LayerTypeRUDP.
func (r *RUDP) LayerType() LayerType { return LayerTypeRUDP }

func decodeRUDP(data []byte) (out DecodeResult, err error) {
	r := &RUDP{
		SYN:          data[0]&0x80 != 0,
		ACK:          data[0]&0x40 != 0,
		EACK:         data[0]&0x20 != 0,
		RST:          data[0]&0x10 != 0,
		NUL:          data[0]&0x08 != 0,
		Version:      data[0] & 0x3,
		HeaderLength: data[1],
		SrcPort:      data[2],
		DstPort:      data[3],
		DataLength:   binary.BigEndian.Uint16(data[4:6]),
		Seq:          binary.BigEndian.Uint32(data[6:10]),
		Ack:          binary.BigEndian.Uint32(data[10:14]),
		Checksum:     binary.BigEndian.Uint32(data[14:18]),
	}
	if r.HeaderLength < 9 {
		err = fmt.Errorf("RUDP packet with too-short header length %d", r.HeaderLength)
		return
	}
	hlen := int(r.HeaderLength) * 2
	r.VariableHeaderArea = data[18:hlen]
	headerData := r.VariableHeaderArea
	switch {
	case r.SYN:
		if len(headerData) != 6 {
			err = fmt.Errorf("RUDP packet invalid SYN header length: %d", len(headerData))
			return
		}
		r.RUDPHeaderSYN = &RUDPHeaderSYN{
			MaxOutstandingSegments: binary.BigEndian.Uint16(headerData[:2]),
			MaxSegmentSize:         binary.BigEndian.Uint16(headerData[2:4]),
			OptionFlags:            binary.BigEndian.Uint16(headerData[4:6]),
		}
	case r.EACK:
		if len(headerData)%4 != 0 {
			err = fmt.Errorf("RUDP packet invalid EACK header length: %d", len(headerData))
			return
		}
		r.RUDPHeaderEACK = &RUDPHeaderEACK{make([]uint32, len(headerData)/4)}
		for i := 0; i < len(headerData); i += 4 {
			r.SeqsReceivedOK[i/4] = binary.BigEndian.Uint32(headerData[i : i+4])
		}
	}
	out.DecodedLayer = r
	out.NextDecoder = decodePayload
	out.RemainingBytes = data[hlen : hlen+int(r.DataLength)]
	out.TransportLayer = r
	return
}

func (r *RUDP) TransportFlow() Flow {
	return Flow{LayerTypeRUDP, string([]byte{r.SrcPort}), string([]byte{r.DstPort})}
}

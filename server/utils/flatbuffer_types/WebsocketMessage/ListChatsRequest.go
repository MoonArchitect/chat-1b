// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package WebsocketMessage

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type ListChatsRequest struct {
	_tab flatbuffers.Table
}

func GetRootAsListChatsRequest(buf []byte, offset flatbuffers.UOffsetT) *ListChatsRequest {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ListChatsRequest{}
	x.Init(buf, n+offset)
	return x
}

func FinishListChatsRequestBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsListChatsRequest(buf []byte, offset flatbuffers.UOffsetT) *ListChatsRequest {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &ListChatsRequest{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedListChatsRequestBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *ListChatsRequest) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ListChatsRequest) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *ListChatsRequest) Uid() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func ListChatsRequestStart(builder *flatbuffers.Builder) {
	builder.StartObject(1)
}
func ListChatsRequestAddUid(builder *flatbuffers.Builder, uid flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(uid), 0)
}
func ListChatsRequestEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}

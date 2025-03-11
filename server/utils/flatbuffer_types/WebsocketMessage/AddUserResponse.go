// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package WebsocketMessage

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type AddUserResponse struct {
	_tab flatbuffers.Table
}

func GetRootAsAddUserResponse(buf []byte, offset flatbuffers.UOffsetT) *AddUserResponse {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &AddUserResponse{}
	x.Init(buf, n+offset)
	return x
}

func FinishAddUserResponseBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsAddUserResponse(buf []byte, offset flatbuffers.UOffsetT) *AddUserResponse {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &AddUserResponse{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedAddUserResponseBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *AddUserResponse) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *AddUserResponse) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *AddUserResponse) ChatId() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *AddUserResponse) UserId() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func AddUserResponseStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func AddUserResponseAddChatId(builder *flatbuffers.Builder, chatId flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(chatId), 0)
}
func AddUserResponseAddUserId(builder *flatbuffers.Builder, userId flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(userId), 0)
}
func AddUserResponseEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}

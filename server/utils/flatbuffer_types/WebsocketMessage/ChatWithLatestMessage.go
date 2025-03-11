// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package WebsocketMessage

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type ChatWithLatestMessage struct {
	_tab flatbuffers.Table
}

func GetRootAsChatWithLatestMessage(buf []byte, offset flatbuffers.UOffsetT) *ChatWithLatestMessage {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ChatWithLatestMessage{}
	x.Init(buf, n+offset)
	return x
}

func FinishChatWithLatestMessageBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsChatWithLatestMessage(buf []byte, offset flatbuffers.UOffsetT) *ChatWithLatestMessage {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &ChatWithLatestMessage{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedChatWithLatestMessageBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *ChatWithLatestMessage) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ChatWithLatestMessage) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *ChatWithLatestMessage) ChatId() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *ChatWithLatestMessage) LatestMessageAt() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ChatWithLatestMessage) MutateLatestMessageAt(n int64) bool {
	return rcv._tab.MutateInt64Slot(6, n)
}

func ChatWithLatestMessageStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func ChatWithLatestMessageAddChatId(builder *flatbuffers.Builder, chatId flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(chatId), 0)
}
func ChatWithLatestMessageAddLatestMessageAt(builder *flatbuffers.Builder, latestMessageAt int64) {
	builder.PrependInt64Slot(1, latestMessageAt, 0)
}
func ChatWithLatestMessageEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}

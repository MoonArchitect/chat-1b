// automatically generated by the FlatBuffers compiler, do not modify

/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */

const flatbuffers = window.flatbuffers

export class ChatWithLatestMessage {
  bb: flatbuffers.ByteBuffer|null = null;
  bb_pos = 0;
  __init(i:number, bb:flatbuffers.ByteBuffer):ChatWithLatestMessage {
  this.bb_pos = i;
  this.bb = bb;
  return this;
}

static getRootAsChatWithLatestMessage(bb:flatbuffers.ByteBuffer, obj?:ChatWithLatestMessage):ChatWithLatestMessage {
  return (obj || new ChatWithLatestMessage()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
}

static getSizePrefixedRootAsChatWithLatestMessage(bb:flatbuffers.ByteBuffer, obj?:ChatWithLatestMessage):ChatWithLatestMessage {
  bb.setPosition(bb.position() + flatbuffers.SIZE_PREFIX_LENGTH);
  return (obj || new ChatWithLatestMessage()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
}

chatId():string|null
chatId(optionalEncoding:flatbuffers.Encoding):string|Uint8Array|null
chatId(optionalEncoding?:any):string|Uint8Array|null {
  const offset = this.bb!.__offset(this.bb_pos, 4);
  return offset ? this.bb!.__string(this.bb_pos + offset, optionalEncoding) : null;
}

latestMessageAt():bigint {
  const offset = this.bb!.__offset(this.bb_pos, 6);
  return offset ? this.bb!.readInt64(this.bb_pos + offset) : BigInt('0');
}

static startChatWithLatestMessage(builder:flatbuffers.Builder) {
  builder.startObject(2);
}

static addChatId(builder:flatbuffers.Builder, chatIdOffset:flatbuffers.Offset) {
  builder.addFieldOffset(0, chatIdOffset, 0);
}

static addLatestMessageAt(builder:flatbuffers.Builder, latestMessageAt:bigint) {
  builder.addFieldInt64(1, latestMessageAt, BigInt('0'));
}

static endChatWithLatestMessage(builder:flatbuffers.Builder):flatbuffers.Offset {
  const offset = builder.endObject();
  return offset;
}

static createChatWithLatestMessage(builder:flatbuffers.Builder, chatIdOffset:flatbuffers.Offset, latestMessageAt:bigint):flatbuffers.Offset {
  ChatWithLatestMessage.startChatWithLatestMessage(builder);
  ChatWithLatestMessage.addChatId(builder, chatIdOffset);
  ChatWithLatestMessage.addLatestMessageAt(builder, latestMessageAt);
  return ChatWithLatestMessage.endChatWithLatestMessage(builder);
}
}

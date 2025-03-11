// automatically generated by the FlatBuffers compiler, do not modify

/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */

const flatbuffers = window.flatbuffers

export class ListMessagesRequest {
  bb: flatbuffers.ByteBuffer|null = null;
  bb_pos = 0;
  __init(i:number, bb:flatbuffers.ByteBuffer):ListMessagesRequest {
  this.bb_pos = i;
  this.bb = bb;
  return this;
}

static getRootAsListMessagesRequest(bb:flatbuffers.ByteBuffer, obj?:ListMessagesRequest):ListMessagesRequest {
  return (obj || new ListMessagesRequest()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
}

static getSizePrefixedRootAsListMessagesRequest(bb:flatbuffers.ByteBuffer, obj?:ListMessagesRequest):ListMessagesRequest {
  bb.setPosition(bb.position() + flatbuffers.SIZE_PREFIX_LENGTH);
  return (obj || new ListMessagesRequest()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
}

chatId():string|null
chatId(optionalEncoding:flatbuffers.Encoding):string|Uint8Array|null
chatId(optionalEncoding?:any):string|Uint8Array|null {
  const offset = this.bb!.__offset(this.bb_pos, 4);
  return offset ? this.bb!.__string(this.bb_pos + offset, optionalEncoding) : null;
}

page():bigint {
  const offset = this.bb!.__offset(this.bb_pos, 6);
  return offset ? this.bb!.readUint64(this.bb_pos + offset) : BigInt('0');
}

static startListMessagesRequest(builder:flatbuffers.Builder) {
  builder.startObject(2);
}

static addChatId(builder:flatbuffers.Builder, chatIdOffset:flatbuffers.Offset) {
  builder.addFieldOffset(0, chatIdOffset, 0);
}

static addPage(builder:flatbuffers.Builder, page:bigint) {
  builder.addFieldInt64(1, page, BigInt('0'));
}

static endListMessagesRequest(builder:flatbuffers.Builder):flatbuffers.Offset {
  const offset = builder.endObject();
  return offset;
}

static createListMessagesRequest(builder:flatbuffers.Builder, chatIdOffset:flatbuffers.Offset, page:bigint):flatbuffers.Offset {
  ListMessagesRequest.startListMessagesRequest(builder);
  ListMessagesRequest.addChatId(builder, chatIdOffset);
  ListMessagesRequest.addPage(builder, page);
  return ListMessagesRequest.endListMessagesRequest(builder);
}
}

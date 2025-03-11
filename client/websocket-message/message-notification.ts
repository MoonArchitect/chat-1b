// automatically generated by the FlatBuffers compiler, do not modify

/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */

const flatbuffers = window.flatbuffers

export class MessageNotification {
  bb: flatbuffers.ByteBuffer|null = null;
  bb_pos = 0;
  __init(i:number, bb:flatbuffers.ByteBuffer):MessageNotification {
  this.bb_pos = i;
  this.bb = bb;
  return this;
}

static getRootAsMessageNotification(bb:flatbuffers.ByteBuffer, obj?:MessageNotification):MessageNotification {
  return (obj || new MessageNotification()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
}

static getSizePrefixedRootAsMessageNotification(bb:flatbuffers.ByteBuffer, obj?:MessageNotification):MessageNotification {
  bb.setPosition(bb.position() + flatbuffers.SIZE_PREFIX_LENGTH);
  return (obj || new MessageNotification()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
}

chatId():string|null
chatId(optionalEncoding:flatbuffers.Encoding):string|Uint8Array|null
chatId(optionalEncoding?:any):string|Uint8Array|null {
  const offset = this.bb!.__offset(this.bb_pos, 4);
  return offset ? this.bb!.__string(this.bb_pos + offset, optionalEncoding) : null;
}

text():string|null
text(optionalEncoding:flatbuffers.Encoding):string|Uint8Array|null
text(optionalEncoding?:any):string|Uint8Array|null {
  const offset = this.bb!.__offset(this.bb_pos, 6);
  return offset ? this.bb!.__string(this.bb_pos + offset, optionalEncoding) : null;
}

userId():string|null
userId(optionalEncoding:flatbuffers.Encoding):string|Uint8Array|null
userId(optionalEncoding?:any):string|Uint8Array|null {
  const offset = this.bb!.__offset(this.bb_pos, 8);
  return offset ? this.bb!.__string(this.bb_pos + offset, optionalEncoding) : null;
}

createdAtMicro():bigint {
  const offset = this.bb!.__offset(this.bb_pos, 10);
  return offset ? this.bb!.readInt64(this.bb_pos + offset) : BigInt('0');
}

msgId():string|null
msgId(optionalEncoding:flatbuffers.Encoding):string|Uint8Array|null
msgId(optionalEncoding?:any):string|Uint8Array|null {
  const offset = this.bb!.__offset(this.bb_pos, 12);
  return offset ? this.bb!.__string(this.bb_pos + offset, optionalEncoding) : null;
}

static startMessageNotification(builder:flatbuffers.Builder) {
  builder.startObject(5);
}

static addChatId(builder:flatbuffers.Builder, chatIdOffset:flatbuffers.Offset) {
  builder.addFieldOffset(0, chatIdOffset, 0);
}

static addText(builder:flatbuffers.Builder, textOffset:flatbuffers.Offset) {
  builder.addFieldOffset(1, textOffset, 0);
}

static addUserId(builder:flatbuffers.Builder, userIdOffset:flatbuffers.Offset) {
  builder.addFieldOffset(2, userIdOffset, 0);
}

static addCreatedAtMicro(builder:flatbuffers.Builder, createdAtMicro:bigint) {
  builder.addFieldInt64(3, createdAtMicro, BigInt('0'));
}

static addMsgId(builder:flatbuffers.Builder, msgIdOffset:flatbuffers.Offset) {
  builder.addFieldOffset(4, msgIdOffset, 0);
}

static endMessageNotification(builder:flatbuffers.Builder):flatbuffers.Offset {
  const offset = builder.endObject();
  return offset;
}

static createMessageNotification(builder:flatbuffers.Builder, chatIdOffset:flatbuffers.Offset, textOffset:flatbuffers.Offset, userIdOffset:flatbuffers.Offset, createdAtMicro:bigint, msgIdOffset:flatbuffers.Offset):flatbuffers.Offset {
  MessageNotification.startMessageNotification(builder);
  MessageNotification.addChatId(builder, chatIdOffset);
  MessageNotification.addText(builder, textOffset);
  MessageNotification.addUserId(builder, userIdOffset);
  MessageNotification.addCreatedAtMicro(builder, createdAtMicro);
  MessageNotification.addMsgId(builder, msgIdOffset);
  return MessageNotification.endMessageNotification(builder);
}
}

// automatically generated by the FlatBuffers compiler, do not modify
/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */
const flatbuffers = window.flatbuffers
export class AddUserResponse {
    constructor() {
        this.bb = null;
        this.bb_pos = 0;
    }
    __init(i, bb) {
        this.bb_pos = i;
        this.bb = bb;
        return this;
    }
    static getRootAsAddUserResponse(bb, obj) {
        return (obj || new AddUserResponse()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
    }
    static getSizePrefixedRootAsAddUserResponse(bb, obj) {
        bb.setPosition(bb.position() + flatbuffers.SIZE_PREFIX_LENGTH);
        return (obj || new AddUserResponse()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
    }
    chatId(optionalEncoding) {
        const offset = this.bb.__offset(this.bb_pos, 4);
        return offset ? this.bb.__string(this.bb_pos + offset, optionalEncoding) : null;
    }
    userId(optionalEncoding) {
        const offset = this.bb.__offset(this.bb_pos, 6);
        return offset ? this.bb.__string(this.bb_pos + offset, optionalEncoding) : null;
    }
    static startAddUserResponse(builder) {
        builder.startObject(2);
    }
    static addChatId(builder, chatIdOffset) {
        builder.addFieldOffset(0, chatIdOffset, 0);
    }
    static addUserId(builder, userIdOffset) {
        builder.addFieldOffset(1, userIdOffset, 0);
    }
    static endAddUserResponse(builder) {
        const offset = builder.endObject();
        return offset;
    }
    static createAddUserResponse(builder, chatIdOffset, userIdOffset) {
        AddUserResponse.startAddUserResponse(builder);
        AddUserResponse.addChatId(builder, chatIdOffset);
        AddUserResponse.addUserId(builder, userIdOffset);
        return AddUserResponse.endAddUserResponse(builder);
    }
}

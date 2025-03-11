// automatically generated by the FlatBuffers compiler, do not modify
/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */
const flatbuffers = window.flatbuffers
export class ListUsersResponse {
    constructor() {
        this.bb = null;
        this.bb_pos = 0;
    }
    __init(i, bb) {
        this.bb_pos = i;
        this.bb = bb;
        return this;
    }
    static getRootAsListUsersResponse(bb, obj) {
        return (obj || new ListUsersResponse()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
    }
    static getSizePrefixedRootAsListUsersResponse(bb, obj) {
        bb.setPosition(bb.position() + flatbuffers.SIZE_PREFIX_LENGTH);
        return (obj || new ListUsersResponse()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
    }
    users(index, optionalEncoding) {
        const offset = this.bb.__offset(this.bb_pos, 4);
        return offset ? this.bb.__string(this.bb.__vector(this.bb_pos + offset) + index * 4, optionalEncoding) : null;
    }
    usersLength() {
        const offset = this.bb.__offset(this.bb_pos, 4);
        return offset ? this.bb.__vector_len(this.bb_pos + offset) : 0;
    }
    chatId(optionalEncoding) {
        const offset = this.bb.__offset(this.bb_pos, 6);
        return offset ? this.bb.__string(this.bb_pos + offset, optionalEncoding) : null;
    }
    static startListUsersResponse(builder) {
        builder.startObject(2);
    }
    static addUsers(builder, usersOffset) {
        builder.addFieldOffset(0, usersOffset, 0);
    }
    static createUsersVector(builder, data) {
        builder.startVector(4, data.length, 4);
        for (let i = data.length - 1; i >= 0; i--) {
            builder.addOffset(data[i]);
        }
        return builder.endVector();
    }
    static startUsersVector(builder, numElems) {
        builder.startVector(4, numElems, 4);
    }
    static addChatId(builder, chatIdOffset) {
        builder.addFieldOffset(1, chatIdOffset, 0);
    }
    static endListUsersResponse(builder) {
        const offset = builder.endObject();
        return offset;
    }
    static createListUsersResponse(builder, usersOffset, chatIdOffset) {
        ListUsersResponse.startListUsersResponse(builder);
        ListUsersResponse.addUsers(builder, usersOffset);
        ListUsersResponse.addChatId(builder, chatIdOffset);
        return ListUsersResponse.endListUsersResponse(builder);
    }
}

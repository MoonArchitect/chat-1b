// automatically generated by the FlatBuffers compiler, do not modify
/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */
const flatbuffers = window.flatbuffers
import { ChatWithLatestMessage } from '../websocket-message/chat-with-latest-message.js';
export class ListChatsResponse {
    constructor() {
        this.bb = null;
        this.bb_pos = 0;
    }
    __init(i, bb) {
        this.bb_pos = i;
        this.bb = bb;
        return this;
    }
    static getRootAsListChatsResponse(bb, obj) {
        return (obj || new ListChatsResponse()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
    }
    static getSizePrefixedRootAsListChatsResponse(bb, obj) {
        bb.setPosition(bb.position() + flatbuffers.SIZE_PREFIX_LENGTH);
        return (obj || new ListChatsResponse()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
    }
    chats(index, obj) {
        const offset = this.bb.__offset(this.bb_pos, 4);
        return offset ? (obj || new ChatWithLatestMessage()).__init(this.bb.__indirect(this.bb.__vector(this.bb_pos + offset) + index * 4), this.bb) : null;
    }
    chatsLength() {
        const offset = this.bb.__offset(this.bb_pos, 4);
        return offset ? this.bb.__vector_len(this.bb_pos + offset) : 0;
    }
    static startListChatsResponse(builder) {
        builder.startObject(1);
    }
    static addChats(builder, chatsOffset) {
        builder.addFieldOffset(0, chatsOffset, 0);
    }
    static createChatsVector(builder, data) {
        builder.startVector(4, data.length, 4);
        for (let i = data.length - 1; i >= 0; i--) {
            builder.addOffset(data[i]);
        }
        return builder.endVector();
    }
    static startChatsVector(builder, numElems) {
        builder.startVector(4, numElems, 4);
    }
    static endListChatsResponse(builder) {
        const offset = builder.endObject();
        return offset;
    }
    static createListChatsResponse(builder, chatsOffset) {
        ListChatsResponse.startListChatsResponse(builder);
        ListChatsResponse.addChats(builder, chatsOffset);
        return ListChatsResponse.endListChatsResponse(builder);
    }
}

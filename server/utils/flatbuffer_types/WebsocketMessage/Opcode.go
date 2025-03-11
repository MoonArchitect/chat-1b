// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package WebsocketMessage

import "strconv"

type Opcode int8

const (
	OpcodeLIST_CHATS_RESPONSE     Opcode = 0
	OpcodeLIST_MESSAGES_RESPONSE  Opcode = 1
	OpcodeCREATE_CHAT_RESPONSE    Opcode = 2
	OpcodeLIST_USERS_RESPONSE     Opcode = 3
	OpcodeCREATE_MESSAGE_RESPONSE Opcode = 4
	OpcodeADD_USER_RESPONSE       Opcode = 5
	OpcodeMESSAGE_NOTIFICATION    Opcode = 6
	OpcodeUSER_LIST_NOTIFICATION  Opcode = 7
	OpcodeCHAT_LIST_NOTIFICATION  Opcode = 8
	OpcodeSEND_MESSAGE_REQUEST    Opcode = 9
	OpcodeLIST_CHATS_REQUEST      Opcode = 10
	OpcodeLIST_MESSAGES_REQUEST   Opcode = 11
	OpcodeLIST_USERS_REQUEST      Opcode = 12
	OpcodeCREATE_CHAT_REQUEST     Opcode = 13
	OpcodeADD_USER_REQUEST        Opcode = 14
)

var EnumNamesOpcode = map[Opcode]string{
	OpcodeLIST_CHATS_RESPONSE:     "LIST_CHATS_RESPONSE",
	OpcodeLIST_MESSAGES_RESPONSE:  "LIST_MESSAGES_RESPONSE",
	OpcodeCREATE_CHAT_RESPONSE:    "CREATE_CHAT_RESPONSE",
	OpcodeLIST_USERS_RESPONSE:     "LIST_USERS_RESPONSE",
	OpcodeCREATE_MESSAGE_RESPONSE: "CREATE_MESSAGE_RESPONSE",
	OpcodeADD_USER_RESPONSE:       "ADD_USER_RESPONSE",
	OpcodeMESSAGE_NOTIFICATION:    "MESSAGE_NOTIFICATION",
	OpcodeUSER_LIST_NOTIFICATION:  "USER_LIST_NOTIFICATION",
	OpcodeCHAT_LIST_NOTIFICATION:  "CHAT_LIST_NOTIFICATION",
	OpcodeSEND_MESSAGE_REQUEST:    "SEND_MESSAGE_REQUEST",
	OpcodeLIST_CHATS_REQUEST:      "LIST_CHATS_REQUEST",
	OpcodeLIST_MESSAGES_REQUEST:   "LIST_MESSAGES_REQUEST",
	OpcodeLIST_USERS_REQUEST:      "LIST_USERS_REQUEST",
	OpcodeCREATE_CHAT_REQUEST:     "CREATE_CHAT_REQUEST",
	OpcodeADD_USER_REQUEST:        "ADD_USER_REQUEST",
}

var EnumValuesOpcode = map[string]Opcode{
	"LIST_CHATS_RESPONSE":     OpcodeLIST_CHATS_RESPONSE,
	"LIST_MESSAGES_RESPONSE":  OpcodeLIST_MESSAGES_RESPONSE,
	"CREATE_CHAT_RESPONSE":    OpcodeCREATE_CHAT_RESPONSE,
	"LIST_USERS_RESPONSE":     OpcodeLIST_USERS_RESPONSE,
	"CREATE_MESSAGE_RESPONSE": OpcodeCREATE_MESSAGE_RESPONSE,
	"ADD_USER_RESPONSE":       OpcodeADD_USER_RESPONSE,
	"MESSAGE_NOTIFICATION":    OpcodeMESSAGE_NOTIFICATION,
	"USER_LIST_NOTIFICATION":  OpcodeUSER_LIST_NOTIFICATION,
	"CHAT_LIST_NOTIFICATION":  OpcodeCHAT_LIST_NOTIFICATION,
	"SEND_MESSAGE_REQUEST":    OpcodeSEND_MESSAGE_REQUEST,
	"LIST_CHATS_REQUEST":      OpcodeLIST_CHATS_REQUEST,
	"LIST_MESSAGES_REQUEST":   OpcodeLIST_MESSAGES_REQUEST,
	"LIST_USERS_REQUEST":      OpcodeLIST_USERS_REQUEST,
	"CREATE_CHAT_REQUEST":     OpcodeCREATE_CHAT_REQUEST,
	"ADD_USER_REQUEST":        OpcodeADD_USER_REQUEST,
}

func (v Opcode) String() string {
	if s, ok := EnumNamesOpcode[v]; ok {
		return s
	}
	return "Opcode(" + strconv.FormatInt(int64(v), 10) + ")"
}

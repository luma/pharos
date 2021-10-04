package protocol

// This package implements the parsing and serialising payloads for the
// the protocol that Pharos uses to communicate with it's clients.
//
// This protocol aims to be
//
// - easy to implement
// - efficient to parse
// - minimize memory usage
// - be human readable
//
// We've stolen many ideas from the Redis protocol (RESP).
//
// - `Command` - A client instruction to Pharos.
// - `Request` - When a client sends a command to a Pharos server.
// - `Response` - When a server sends a command response to a client.
// - `Update` - A notification of a change to a single key. These are
//              sent from a Pharos server to it's clients.
//
// === Client Commands
//
// - `QUIT` - indicates that the client wishes to quit and that the server can
//					  close the connection
// - `PING` - PING! Server will respond with pong
// - `SET`  - The client wishes to update a key to the provided value
//
// === General Syntax
//
// - lines are `\r\n` delimited
// - Client commands are indicates using their human-readable name (e.g. 'QUIT')
// - Command names are case sensitive and should be uppercase
//
// As the server will send key updates whenever they are ready, key updates from
// the server can interleave with commands, or command replies, from the client.
// This would make parsing more difficult for the client so the client request/response
// exchanges are prefixed with a request ID.
//
// The request ID is treated as a 32bit binary blob by the server so the client
// can construct it however it likes.
//
// For example
//   ```
//     <reqID>PING\r\n
//     <reqID>PONG\r\n
//   ```
//
// The PING is the command sent to the client, the PONG is the reply from the server.
// THe server includes the request ID in it's response so the client can associate
// the reply with the right request.
//
// Note: requests and their response can interleave with other requests/responses or
//       updates, but a single request, response, or update is atomic. Meaning you will
//			 never receive half of a response, then an entire update, then the rest of
//       the response.
//
// === Error responses
//
//   ```
//     <reqID>PING\r\n
//     <reqID>ERR <errMessage>\r\n
//   ```
//
// Where `<errMessage>` is a human readable string
//
// === QUIT
//
//  ```
//    > <reqID>QUIT\r\n
//    < <reqID>OK\r\n
//  ```
//
// === PING
//
//  ```
//    > <reqID>PING\r\n
//    < <reqID>OK\r\n
//  ```
//
//
// === SET
//
//  ```
//    > <reqID>SET <key>\r\n
//    > <value>\r\n
//    < <reqID>OK\r\n
//  ```
//
// === Key updates
//
// Whenever keys are updated by clients the servers will push the updated keys
// to every client. Key updates are the only communication is that isn't a
// request/response exchange initiated by the client. Hence it's different
// in several ways.
//
// - Updates will never include request IDs, as they aren't initiated by the client
// - The are prefixed with `$`
//
// The syntax of a full update is as follows
//
//   ```
//   $<key>\n\n
//   <update>\n\n
//   ```
//
// The first line says that "this is a update of key <key>". The second line is the
// encoded value of the key.
//
// ==== Update Value encoding
//
// TODO(rolly) but JSON for now...
//

package iterator

import (
	"encoding/binary"
	"fmt"
	"github.com/sigurn/crc8"
	"io"
)

// Record Header Structure (min 10 bytes, max 13 bytes per record):
// +-----+-----+-----+-----+-----+-----+-----+-----+
// | ERR | HiQ | HiR |       Call Namespace        |
// +-----+-----+-----+-----+-----+-----+-----+-----+
// |      Call Method      |     Query Size Hi     |
// +-----+-----+-----+-----+-----+-----+-----+-----+
// |        Query Size Hi (skip if HiQ = 0)        |
// +-----+-----+-----+-----+-----+-----+-----+-----+
// |                 Query Size Lo                 |
// +-----+-----+-----+-----+-----+-----+-----+-----+
// |               Response Size Hi                |
// |     (16 bits; skip if HiR = 0 OR ERR = 1)     |
// +-----+-----+-----+-----+-----+-----+-----+-----+
// |   Response Size Lo OR Error Code if ERR = 1   |
// |                  (16 bits)                    |
// +-----+-----+-----+-----+-----+-----+-----+-----+
// |                                               |
// |            Response Block Number              |
// |                  (32 bits)                    |
// |                                               |
// +-----+-----+-----+-----+-----+-----+-----+-----+
// |                 CRC8 Checksum                 |
// +-----+-----+-----+-----+-----+-----+-----+-----+

// maxQuerySizeAllowed represents the maximal size of recordable query; 4 bits (Hi) + 8 bits (HiQ = 1) + 8 bits (Lo)
const maxQuerySizeAllowed = 0xFFFFF

// maxShortQuery represents the longest request query still considered as short (4 bits Hi + 8 bits Lo).
const maxShortQuery = 0xFFF

// maxShortResponse represents the longest response payload still considered as short (16 bits uint).
const maxShortResponse = 0xFFFF

// Header represents single record header on a virtual recording tape represented by a Reader/Writer.
type Header struct {
	namespace      byte
	method         byte
	isError        bool
	isLongQuery    bool
	isLongResult   bool
	querySize      int32
	resultCodeSize int32 // also used for error code; see ERR flag
	blockID        uint64
}

// namespaceDictionary represents a dictionary of call namespace for encoding.
// Each namespace is supposed to be marked by its own bit to allow multi-namespace filtering on the reader.
// Unlisted namespaces are not recorded.
var namespaceDictionary = map[string]byte{
	"eth": 1 << 0,
	"ftm": 1 << 0, // ftm is a copy of the eth namespace
}

// methodDictionary represents a dictionary of methods by namespace for encoding.
// Unlisted methods are not recorded.
var methodDictionary = map[byte]map[string]byte{
	1 << 0: {
		/* eth+ftm namespaces */
		"call":                1,
		"estimateGas":         2,
		"getBalance":          3,
		"getCode":             4,
		"getStorageAt":        5,
		"getTransactionCount": 6,
		"getLogs":             7,
	},
}

// checksumTable is the table used to calculate the header checksum.
var checksumTable = crc8.MakeTable(crc8.CRC8_CDMA2000)

// SetMethod sets a call namespace and method into the header.
func (h *Header) SetMethod(namespace string, method string) error {
	var ok bool

	h.namespace, ok = namespaceDictionary[namespace]
	if !ok {
		return fmt.Errorf("namespace '%s' not recorded", namespace)
	}

	h.method, ok = methodDictionary[h.namespace][method]
	if !ok {
		return fmt.Errorf("method '%s' of namespace '%s' not recorded", method, namespace)
	}

	return nil
}

// Namespace returns the namespace set in the header. An error is returned if no namespace has been set.
func (h *Header) Namespace() (string, error) {
	if h.namespace == 0 {
		return "", fmt.Errorf("namespace not initialized")
	}

	for n, i := range namespaceDictionary {
		if h.namespace == i {
			return n, nil
		}
	}

	return "", fmt.Errorf("unknown namespace set")
}

// Method returns the method name set in the header. An error is returned if no namespace has been set.
func (h *Header) Method() (string, error) {
	if h.namespace == 0 || h.method == 0 {
		return "", fmt.Errorf("namespace or method not initialized")
	}

	for n, i := range methodDictionary[h.namespace] {
		if h.method == i {
			return n, nil
		}
	}

	return "", fmt.Errorf("unknown method set")
}

// SetBlockID configures the ID of a block context the query was executed under.
func (h *Header) SetBlockID(id uint64) {
	h.blockID = id
}

// BlockID returns the block ID of the record.
func (h *Header) BlockID() uint64 {
	return h.blockID
}

// SetQueryLength configures the query length.
func (h *Header) SetQueryLength(ql int) error {
	// we have to skip queries too big to be stored
	if ql > maxQuerySizeAllowed {
		return fmt.Errorf("query too big; expected max %d bytes, received %d", maxQuerySizeAllowed, ql)
	}

	h.querySize = int32(ql)
	h.isLongQuery = h.querySize > maxShortQuery // short query is 8 bits (Lo) + 4 bits (Hi)
	return nil
}

// QueryLength returns expected size of the query.
func (h *Header) QueryLength() int32 {
	return h.querySize
}

// SetError configures the header to describe error response to the given query call.
func (h *Header) SetError(errCode int) {
	h.isError = true
	h.resultCodeSize = int32(errCode)
	h.isLongResult = false
}

// ErrorCode returns the error code currently set in the header.
// Zero is returned for no error.
func (h *Header) ErrorCode() int {
	if !h.isError {
		return 0
	}
	return int(h.resultCodeSize)
}

// IsError returns the error status of the Header.
func (h *Header) IsError() bool {
	return h.isError
}

// SetResponseLength configures the response length.
func (h *Header) SetResponseLength(rl int) {
	h.isError = false
	h.resultCodeSize = int32(rl)
	h.isLongResult = h.resultCodeSize > maxShortResponse
}

// ResponseLength returns expected size of the response.
// Returns zero value for error responses.
func (h *Header) ResponseLength() int32 {
	if h.isError {
		return 0
	}
	return h.resultCodeSize
}

// WriteTo writes the current header into the given Writer target.
func (h *Header) WriteTo(out io.Writer) (int64, error) {
	hdr := make([]byte, 13)

	offset := h.codeQuery(hdr)

	if h.isError {
		offset += h.codeError(hdr, offset)
	} else {
		offset += h.codeResponse(hdr, offset)
	}

	// append the block number
	binary.BigEndian.PutUint32(hdr[offset:offset+4], uint32(h.blockID))

	// add the CRC8/CDMA2000 checksum
	hdr[offset+4] = crc8.Checksum(hdr[:offset+4], checksumTable)

	n, e := out.Write(hdr[:offset+5])
	return int64(n), e
}

// codeQuery encodes query part of the header into the given buffer returning the number of bytes used.
func (h *Header) codeQuery(hdr []byte) int {
	// namespace (5 bits)
	hdr[0] = h.namespace & 0x1F

	// add query size; 12 bits (4kB) for short, or 20 bits (~1MB) for long signaled by HiQ flag
	if !h.isLongQuery {
		hdr[1] |= (h.method&0xF)<<4 | byte(h.querySize>>8)&0xF
		hdr[2] = byte(h.querySize & 0xFF)
		return 3 // short query, omit the Size Hi byte
	}

	hdr[0] |= 1 << 6
	hdr[1] = (h.method&0xF)<<4 | byte(h.querySize>>16)&0xF
	hdr[2] = byte(h.querySize >> 8)
	hdr[3] = byte(h.querySize)
	return 4 // long query, the Size Hi byte is present
}

// codeError encodes error response part of the header into the given buffer returning number of bytes used.
// Note: Error response uses Response Size Lo field to store the error code.
func (h *Header) codeError(hdr []byte, offset int) int {
	hdr[0] |= 1 << 7
	binary.BigEndian.PutUint16(hdr[offset:offset+2], uint16(h.resultCodeSize))
	return 2
}

// codeResponse encodes regular non-error response part of the header into the given buffer
// returning number of bytes used.
// Response Size can use 16 bits (64kB) for short, or 32 bits (4GB) for long signaled by HiR flag.
func (h *Header) codeResponse(hdr []byte, offset int) int {
	if h.isLongResult {
		hdr[0] |= 1 << 5
		binary.BigEndian.PutUint32(hdr[offset:offset+4], uint32(h.resultCodeSize))
		return 4
	}

	binary.BigEndian.PutUint16(hdr[offset:offset+2], uint16(h.resultCodeSize))
	return 2
}

// ReadFrom reads the header from the given reader.
func (h *Header) ReadFrom(r io.Reader) (int64, error) {
	// read the header into buffer and pre-decode flags
	hdr, err := h.readFrom(r)
	if err != nil {
		return int64(len(hdr)), err
	}

	h.decodeFields(hdr)
	return int64(len(hdr)), nil
}

// readFrom reads the header from Reader and pre-decodes internal flags.
func (h *Header) readFrom(r io.Reader) ([]byte, error) {
	hdr := make([]byte, 13)
	var err error

	// read the first byte to get the idea of how long the header is
	_, err = r.Read(hdr[:1])
	if err != nil {
		return hdr[0:0], err
	}

	h.isError = hdr[0]&(1<<7) > 0
	h.isLongQuery = hdr[0]&(1<<6) > 0
	h.isLongResult = hdr[0]&(1<<5) > 0

	// calculate the total header size based on received flags
	size := 10
	if h.isLongQuery {
		size += 1
	}
	if h.isLongResult {
		size += 2
	}

	n := 1 // we already have the first byte
	var i int
	for n < size {
		i, err = r.Read(hdr[n:size])
		n += i

		if err != nil {
			break
		}
	}

	// check if what we got make sense
	crc := crc8.Checksum(hdr[:size-1], checksumTable)
	if crc != hdr[size-1] {
		return nil, fmt.Errorf("invalid record header checksum")
	}

	return hdr[:size], nil
}

// decodeFields decodes data fields from the given loaded binary header.
func (h *Header) decodeFields(hdr []byte) {
	h.namespace = hdr[0] & 0x1F
	h.method = hdr[1] >> 4

	var offset int
	if h.isLongQuery {
		h.querySize = int32(hdr[1]&0xF)<<16 | int32(hdr[2])<<8 | int32(hdr[3])
		offset = 4
	} else {
		h.querySize = int32(hdr[1]&0xF)<<8 | int32(hdr[2])
		offset = 3
	}

	if h.isLongResult {
		h.resultCodeSize = int32(binary.BigEndian.Uint32(hdr[offset : offset+4]))
		offset += 4
	} else {
		if h.isError {
			// double conversion flips unsigned to signed int first before expanding the range
			h.resultCodeSize = int32(int16(binary.BigEndian.Uint16(hdr[offset : offset+2])))
		} else {
			h.resultCodeSize = int32(binary.BigEndian.Uint16(hdr[offset : offset+2]))
		}
		offset += 2
	}

	h.blockID = uint64(binary.BigEndian.Uint32(hdr[offset : offset+4]))
}

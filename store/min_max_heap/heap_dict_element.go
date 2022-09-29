package min_max_heap

import "pandora-pay/helpers"

type HeapDictElement struct {
	HashmapKey []byte //hashmap key
	Position   uint64
}

func (this *HeapDictElement) IsDeletable() bool {
	return false
}

func (this *HeapDictElement) SetKey(key []byte) {
	this.HashmapKey = key
}

func (this *HeapDictElement) SetIndex(index uint64) { //not indexable
}

func (this *HeapDictElement) GetIndex() uint64 { //not indexable
	return 0
}

func (this *HeapDictElement) Validate() error {
	return nil
}

func (this *HeapDictElement) Serialize(w *helpers.BufferWriter) {
	w.WriteUvarint(this.Position)
}

func (this *HeapDictElement) Deserialize(r *helpers.BufferReader) (err error) {
	this.Position, err = r.ReadUvarint()
	return
}

package zoom

// import "github.com/blevesearch/bleve"

// TODO: remove entry from index, have multiple entries per key, delete/add to multiple entries
/*
	TODO change index the following way:

	1. to delete an entry set the key to "", if there is an empty key before the to be delete, the first empty key in order
	   will have the correct number of following empty blocks.
	   if the entry has no empty key before, it will have the number of following empty keys

	2. if an entry is added, there will be a lookup, if a value of the key is already there,
	   if so each following value of the same key is collected until key changes and the keys are
	   deleted in place (set to ""). then the collected values and the new values will be written in order
	   at some free place

	3. to get a value, if the key is found, all following values having the same key will be collected

	4. there is a max number of values for an index that will be respected when adding
		 if max is reached, adding is an error, instead entries must be deleted before new can be added
		 a max value of 1 is a unique index for the key
		 a max value of -1 is no limit for max entries
		 a max value of 0 should never be there


*/

/*
// mapping := bleve.NewIndexMapping()
func CreateIndex(shard, path string) (Index2, error) {
	mapping := bleve.NewIndexMapping()
	idx, err := bleve.New(path, mapping)
	return Index2{idx: idx}, err
}
*/

// index.Index(message.Id, message)

// index, _ := bleve.Open("example.bleve")

/*
query := bleve.NewQueryStringQuery("bleve")
    searchRequest := bleve.NewSearchRequest(query)
    searchResult, _ := index.Search(searchRequest)
*/
/*
type Index struct {
	KeyWidth   int
	ValueWidth int
	Shard      string
	Path       string
	MaxNumVals int
	data       []byte
}

func NewIndex(shard, path string, keyWidth, valWidth, maxNumVal int) *Index {
	return &Index{
		KeyWidth:   keyWidth,
		ValueWidth: valWidth,
		Shard:      shard,
		Path:       path,
		MaxNumVals: maxNumVal,
	}
}

func (i *Index) Save(st Store) error {
	// fmt.Println(string(i.data))
	return st.SaveIndex(i.Path, i.Shard, bytes.NewReader(i.data))
}

func (i *Index) Load(st Store) error {
	return st.GetIndex(i.Path, i.Shard, func(rd io.Reader) error {
		data, err := ioutil.ReadAll(rd)
		if err == nil {
			i.data = data
		}
		return err
	})
}

//func (i *Index) writeString(bf *bytes.Buffer, width int, str string) {
func (i *Index) writeString(wr io.Writer, width int, str string) error {
	l := len(str)
	err := binary.Write(wr, binary.LittleEndian, uint8(l))
	if err != nil {
		return err
	}
	rest := width - l
	for i := rest; i > 0; i-- {
		str += " "
	}
	_, err = wr.Write([]byte(str))
	return err
}
*/

/*
	The basic idea is that the index should be a bytearray that has the following structure

	[len of key in bytes][key][pad until keywidth][len of value][pad until valuewidth][len of key in bytes]...

	where len of key in bytes is 1 byte (1 to 255) and len of value is 1 byte (1 to 255) such that
	one entry is always keywidth+valuewidth+2bytes long where keywidth and valuewidth max at 255 such that
	one entry is 255+255+2 = 512 bytes at most

	so that we can scan the file in blocks of keywidth+valuewidth+2bytes

*/

/*
func (i *Index) Add(key string, value string) (err error) {
	var bf bytes.Buffer
	err = i.add(key, value, &bf)
	if err == nil {
		i.data = append(i.data, bf.Bytes()...)
	}
	return
}

func (i *Index) add(key string, value string, wr io.Writer) (err error) {
	if len(key) > i.KeyWidth {
		panic("key to long " + key)
	}
	if len(value) > i.ValueWidth {
		panic("value to long " + value)
	}

	err = i.writeString(wr, i.KeyWidth, key)
	if err != nil {
		return err
	}

	return i.writeString(wr, i.ValueWidth, value)

}

func (i *Index) Find(key string) (value string, err error) {
	bf := bytes.NewBuffer(i.data)
	return i.find(key, bf)
}

// TODO: take a writeseeker in order to write at a specific position
func (i *Index) remove(key string) (err error) {
*/
/*
	buf := make([]byte, i.KeyWidth+i.ValueWidth+2)
	var block int
	var n int
	for {
		n, err = rd.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}

		bff := bytes.NewBuffer(buf)
		var l uint8

		err = binary.Read(bff, binary.LittleEndian, &l)
		if err != nil {
			return "", err
		}

		k := string(buf[1 : l+1])

		if k == key {
			var l2 uint8
			bff = bytes.NewBuffer(buf[i.KeyWidth+1 : i.KeyWidth+2])
			err = binary.Read(bff, binary.LittleEndian, &l2)
			if err != nil {
				return "", err
			}
			return string(buf[i.KeyWidth+2 : i.KeyWidth+2+int(l2)]), nil
		}
		block++
	}
	return "", nil
*/
/*	return
}

func (i *Index) find(key string, rd io.Reader) (value string, err error) {

	buf := make([]byte, i.KeyWidth+i.ValueWidth+2)
	var n int
	for {
		n, err = rd.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}

		bff := bytes.NewBuffer(buf)
		var l uint8

		err = binary.Read(bff, binary.LittleEndian, &l)
		if err != nil {
			return "", err
		}

		k := string(buf[1 : l+1])

		if k == key {
			var l2 uint8
			bff = bytes.NewBuffer(buf[i.KeyWidth+1 : i.KeyWidth+2])
			err = binary.Read(bff, binary.LittleEndian, &l2)
			if err != nil {
				return "", err
			}
			return string(buf[i.KeyWidth+2 : i.KeyWidth+2+int(l2)]), nil
		}
	}
	return "", nil
}

var (
	StringWidth = 255
	IntWidth    = 10 // TODO: lookup
	// 7196aced-8418-4412-b0ce-4994998aa73f
	UUIDWidth = 36
)
*/

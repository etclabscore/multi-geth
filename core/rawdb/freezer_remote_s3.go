// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

type freezerRemoteS3 struct {
	session *session.Session
	service *s3.S3

	namespace string
	quit      chan struct{}
	mu        sync.Mutex

	readMeter  metrics.Meter // Meter for measuring the effective amount of data read
	writeMeter metrics.Meter // Meter for measuring the effective amount of data written
	sizeGauge  metrics.Gauge // Gauge for tracking the combined size of all freezer tables

	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader

	frozen          *uint64 // the length of the frozen blocks (next appended must == val)
	objectGroupSize uint64  // how many blocks to include in a single S3 object

	retrieved map[uint64]AncientObjectS3
	cache     map[uint64]AncientObjectS3
	cacheS    []uint64

	log log.Logger
}

type AncientObjectS3 struct {
	Hash       common.Hash                `json:"hash"`
	Header     *types.Header              `json:"header"`
	Body       *types.Body                `json:"body"`
	Receipts   []*types.ReceiptForStorage `json:"receipts"`
	Difficulty *big.Int                   `json:"difficulty"`
}

func NewAncientObjectS3(hashB, headerB, bodyB, receiptsB, difficultyB []byte) (*AncientObjectS3, error) {
	var err error

	hash := common.BytesToHash(hashB)

	header := &types.Header{}
	err = rlp.DecodeBytes(headerB, header)
	if err != nil {
		return nil, err
	}
	body := &types.Body{}
	err = rlp.DecodeBytes(bodyB, body)
	if err != nil {
		return nil, err
	}
	receipts := []*types.ReceiptForStorage{}
	err = rlp.DecodeBytes(receiptsB, &receipts)
	if err != nil {
		return nil, err
	}
	difficulty := new(big.Int)
	err = rlp.DecodeBytes(difficultyB, difficulty)
	if err != nil {
		return nil, err
	}
	return &AncientObjectS3{
		Hash:       hash,
		Header:     header,
		Body:       body,
		Receipts:   receipts,
		Difficulty: difficulty,
	}, nil
}

func (o *AncientObjectS3) RLPBytesForKind(kind string) []byte {
	switch kind {
	case freezerHashTable:
		return o.Hash.Bytes()
	case freezerHeaderTable:
		b, err := rlp.EncodeToBytes(o.Header)
		if err != nil {
			log.Crit("Failed to RLP encode block header", "err", err)
		}
		return b
	case freezerBodiesTable:
		b, err := rlp.EncodeToBytes(o.Body)
		if err != nil {
			log.Crit("Failed to RLP encode block body", "err", err)
		}
		return b
	case freezerReceiptTable:
		b, err := rlp.EncodeToBytes(o.Receipts)
		if err != nil {
			log.Crit("Failed to RLP encode block receipts", "err", err)
		}
		return b
	case freezerDifficultyTable:
		b, err := rlp.EncodeToBytes(o.Difficulty)
		if err != nil {
			log.Crit("Failed to RLP encode block difficulty", "err", err)
		}
		return b
	default:
		panic(fmt.Sprintf("unknown kind: %s", kind))
	}
}

func awsKeyBlock(number uint64) string {
	// Keep blocks in a dir.
	// This namespaces the resource, separating it from the 'index-marker' object.
	return fmt.Sprintf("blocks/%09d.json", number)
}

func (f *freezerRemoteS3) objectKeyForN(n uint64) string {
	return awsKeyBlock((n / f.objectGroupSize) * f.objectGroupSize) // 0, 32, 64, 96, ...
}

// TODO: this is superfluous now; bucket names must be user-configured
func (f *freezerRemoteS3) bucketName() string {
	return fmt.Sprintf("%s", f.namespace)
}

func (f *freezerRemoteS3) initializeBucket() error {
	bucketName := f.bucketName()
	start := time.Now()
	f.log.Info("Creating bucket if not exists", "name", bucketName)
	result, err := f.service.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(f.bucketName()),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists, s3.ErrCodeBucketAlreadyOwnedByYou:
				f.log.Debug("Bucket exists", "name", bucketName)
				return nil
			}
		}
		return err
	}
	err = f.service.WaitUntilBucketExists(&s3.HeadBucketInput{
		Bucket: aws.String(f.bucketName()),
	})
	if err != nil {
		return err
	}
	f.log.Info("Bucket created", "name", bucketName, "result", result.String(), "elapsed", time.Since(start))
	return nil
}

func (f *freezerRemoteS3) initCache(n uint64) error {
	f.log.Info("Initializing cache", "n", n)

	key := f.objectKeyForN(n)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := f.downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(f.bucketName()),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return errOutOfBounds
			}
		}
		f.log.Error("Download error", "method", "initCache", "error", err, "key", key)
		return err
	}
	target := []AncientObjectS3{}
	err = json.Unmarshal(buf.Bytes(), &target)
	if err != nil {
		return err
	}
	for _, v := range target {
		n := v.Header.Number.Uint64()
		f.cacheS = append(f.cacheS, n)
		f.cache[n] = v
	}
	f.log.Info("Finished initializing cache", "n", n, "size", len(f.cache))
	if f.cacheS[0] % f.objectGroupSize != 0 {
		panic(fmt.Sprintf("cache does not begin at mod: n=%d (mod=%d)", f.cacheS[0], f.cacheS[0] % f.objectGroupSize))
	}
	return nil
}

// newFreezer creates a chain freezer that moves ancient chain data into
// append-only flat file containers.
func newFreezerRemoteS3(namespace string, readMeter, writeMeter metrics.Meter, sizeGauge metrics.Gauge) (*freezerRemoteS3, error) {
	var err error

	freezerGroups := uint64(32)
	if v := os.Getenv("GETH_FREEZER_S3_GROUP_OBJECTS"); v != "" {
		i, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		freezerGroups = i
	}
	f := &freezerRemoteS3{
		namespace:       namespace,
		quit:            make(chan struct{}),
		readMeter:       readMeter,
		writeMeter:      writeMeter,
		sizeGauge:       sizeGauge,
		objectGroupSize: freezerGroups,
		retrieved:       make(map[uint64]AncientObjectS3),
		cache:           make(map[uint64]AncientObjectS3),
		cacheS: []uint64{},
		log:             log.New("remote", "s3"),
	}

	/*
		By default NewSession will only load credentials from the shared credentials file (~/.aws/credentials).
		If the AWS_SDK_LOAD_CONFIG environment variable is set to a truthy value the Session will be created from the
		configuration values from the shared config (~/.aws/config) and shared credentials (~/.aws/credentials) files.
		Using the NewSessionWithOptions with SharedConfigState set to SharedConfigEnable will create the session as if the
		AWS_SDK_LOAD_CONFIG environment variable was set.
		> https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
	*/
	f.session, err = session.NewSession()
	if err != nil {
		f.log.Info("Session", "err", err)
		return nil, err
	}
	f.log.Info("New session", "region", f.session.Config.Region)
	f.service = s3.New(f.session)

	// Create buckets per the schema, where each bucket is prefixed with the namespace
	// and suffixed with the schema Kind.
	err = f.initializeBucket()
	if err != nil {
		return f, err
	}

	f.uploader = s3manager.NewUploader(f.session)
	f.uploader.Concurrency = 10

	f.downloader = s3manager.NewDownloader(f.session)

	n, _ := f.Ancients()
	f.frozen = &n

	if n > 0 {
		err = f.initCache(n)
		if err != nil {
			return f, err
		}
	}

	return f, nil
}

// Close terminates the chain freezer, unmapping all the data files.
func (f *freezerRemoteS3) Close() error {
	f.quit <- struct{}{}
	// I don't see any Close, Stop, or Quit methods for the AWS service.
	return nil
}

// HasAncient returns an indicator whether the specified ancient data exists
// in the freezer.
func (f *freezerRemoteS3) HasAncient(kind string, number uint64) (bool, error) {
	v, err := f.Ancient(kind, number)
	if err != nil {
		return false, err
	}

	return v != nil, nil
}

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (f *freezerRemoteS3) Ancient(kind string, number uint64) ([]byte, error) {
	if atomic.LoadUint64(f.frozen) <= number {
		return nil, nil
	}
	if v, ok := f.cache[number]; ok {
		return v.RLPBytesForKind(kind), nil
	}
	if v, ok := f.retrieved[number]; ok {
		return v.RLPBytesForKind(kind), nil
	}


	// Take from remote
	key := f.objectKeyForN(number)
	f.log.Info("Downloading ancient(s)", "kind", kind, "number", number, "key", key)

	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := f.downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(f.bucketName()),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return nil, errOutOfBounds
			}
		}
		f.log.Error("Download error", "method", "Ancient", "error", err, "kind", kind, "key", key, "number", number)
		return nil, err
	}
	target := []AncientObjectS3{}
	err = json.Unmarshal(buf.Bytes(), &target)
	if err != nil {
		return nil, err
	}
	f.log.Info("Finished downloading ancient(s)", "len", len(target))
	f.retrieved = map[uint64]AncientObjectS3{}
	o := &AncientObjectS3{}
	for _, v := range target {
		n := v.Header.Number.Uint64()
		f.retrieved[n] = v
		if n == number {
			*o = v
		}
	}
	if o.Hash == (common.Hash{}) {
		j, _ := json.MarshalIndent(target, "", "    ")
		fmt.Println("number", number, "kind", kind, "key", key)
		fmt.Println(string(j))
		panic("bad")
	}
	return o.RLPBytesForKind(kind), nil
}

// Ancients returns the length of the frozen items.
func (f *freezerRemoteS3) Ancients() (uint64, error) {
	if f.frozen != nil {
		return atomic.LoadUint64(f.frozen), nil
	}
	f.log.Info("Retrieving ancients number")
	result, err := f.service.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(f.bucketName()),
		Key:    aws.String("index-marker"),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return 0, nil
			}
		}
		f.log.Error("GetObject error", "method", "Ancients", "error", err)
		return 0, err
	}
	contents, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return 0, err
	}
	i, err := strconv.ParseUint(string(contents), 10, 64)
	f.log.Info("Finished retrieving ancients num", "n", i)
	return i, err
}

// AncientSize returns the ancient size of the specified category.
func (f *freezerRemoteS3) AncientSize(kind string) (uint64, error) {
	// AWS Go-SDK doesn't support this in a convenient way.
	// This would require listing all objects in the bucket and summing their sizes.
	// This method is only used in the InspectDatabase function, which isn't that
	// important.
	return 0, errNotSupported
}

func (f *freezerRemoteS3) setIndexMarker(number uint64) error {
	f.log.Info("Setting index marker", "number", number)
	numberStr := strconv.FormatUint(number, 10)
	reader := bytes.NewReader([]byte(numberStr))
	_, err := f.service.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(f.bucketName()),
		Key:    aws.String("index-marker"),
		Body:   reader,
	})
	return err
}

// AppendAncient injects all binary blobs belong to block at the end of the
// append-only immutable table files.
//
// Notably, this function is lock free but kind of thread-safe. All out-of-order
// injection will be rejected. But if two injections with same number happen at
// the same time, we can get into the trouble.
func (f *freezerRemoteS3) AppendAncient(number uint64, hash, header, body, receipts, td []byte) (err error) {

	f.mu.Lock()
	defer f.mu.Unlock()

	o, err := NewAncientObjectS3(hash, header, body, receipts, td)
	if err != nil {
		return err
	}
	f.cache[number] = *o
	f.cacheS = append(f.cacheS, number)

	if f.cacheS[0] % f.objectGroupSize != 0 {
		panic(fmt.Sprintf("cache does not begin at mod: n=%d (mod=%d)", f.cacheS[0], f.cacheS[0] % f.objectGroupSize))
	}

	atomic.AddUint64(f.frozen, 1)

	return nil
}

// Truncate discards any recent data above the provided threshold number.
// TODO@meowsbits: handle pagination.
//   ListObjects will only return the first 1000. Need to implement pagination.
//   Also make sure that the Marker is working as expected.
func (f *freezerRemoteS3) TruncateAncients(items uint64) error {

	f.mu.Lock()
	defer f.mu.Unlock()

	var err error

	log.Info("Truncating ancients", "cacheS", f.cacheS, "items", items)

	if f.cacheS[0] % f.objectGroupSize != 0 {
		panic(fmt.Sprintf("cache does not begin at mod: n=%d (mod=%d)", f.cacheS[0], f.cacheS[0] % f.objectGroupSize))
	}

	cachedAt := -1
	for i, v := range f.cacheS {
		if v >= items {
			delete(f.cache, v)
		}
		if v == items {
			cachedAt = i
		}
	}
	if cachedAt >= 0 {
		f.cacheS = f.cacheS[:cachedAt]
		log.Info("Truncated cached ancients", "cacheS", f.cacheS)
		err = f.setIndexMarker(items)
		if err != nil {
			return err
		}
		atomic.StoreUint64(f.frozen, items)
		return nil
	}

	if f.cacheS[0] % f.objectGroupSize != 0 {
		panic(fmt.Sprintf("cache does not begin at mod: n=%d (mod=%d)", f.cacheS[0], f.cacheS[0] % f.objectGroupSize))
	}

	n := atomic.LoadUint64(f.frozen)

	// First, download the latest group object into a temporary cache.
	if items % f.objectGroupSize != 0 {
		key := f.objectKeyForN(items)
		buf := aws.NewWriteAtBuffer([]byte{})
		_, err = f.downloader.Download(buf, &s3.GetObjectInput{
			Bucket: aws.String(f.bucketName()),
			Key:    aws.String(key),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case s3.ErrCodeNoSuchKey:
					return errOutOfBounds
				}
			}
			f.log.Error("Download error", "method", "TruncateAncients", "error", err, "key", key, "items", items)
			return err
		}
		target := []AncientObjectS3{}
		err = json.Unmarshal(buf.Bytes(), &target)
		if err != nil {
			return err
		}
		f.cache = map[uint64]AncientObjectS3{}
		f.cacheS = []uint64{}
		for _, t := range target {
			n := t.Header.Number.Uint64()
			if n < items {
				f.cache[n] = t
				f.cacheS = append(f.cacheS, n)
			}
		}
		sort.Slice(f.cacheS, func(i, j int) bool {
			return f.cacheS[i] < f.cacheS[j]
		})
	} else {
		f.cache = map[uint64]AncientObjectS3{}
		f.cacheS = []uint64{}
	}

	// Now truncate all remote data from the latest grouped object and beyond.
	// Noting that this can remove data from the remote that is actually antecedent to the
	// desired truncation level, since we have to use the groups.
	// That's why we pulled the object into the cache first; on the next Sync, the un-truncated
	// blocks will be pushed back up to the remote.
	f.log.Info("Truncating ancients", "ancients", n, "target", items, "delta", n-items)
	start := time.Now()

	list := &s3.ListObjectsInput{
		Bucket: aws.String(f.bucketName()),
		Marker: aws.String(f.objectKeyForN(items)),
	}
	iter := s3manager.NewDeleteListIterator(f.service, list)
	batcher := s3manager.NewBatchDeleteWithClient(f.service)
	if err := batcher.Delete(aws.BackgroundContext(), iter); err != nil {
		return err
	}

	if len(f.cache) > 0 {
		err = f.pushCache()
		if err != nil {
			return err
		}
	}

	err = f.setIndexMarker(items)
	if err != nil {
		return err
	}
	atomic.StoreUint64(f.frozen, items)

	f.log.Info("Finished truncating ancients", "elapsed", time.Since(start))
	return nil
}

func sliceIndexOf(sl []uint64, n uint64) int {
	for i, s := range sl {
		if s == n {
			return i
		}
	}
	return -1
}

func (f *freezerRemoteS3) pushCache() error {
	if len(f.cacheS) == 0 {
		return nil
	}

	if f.cacheS[0] % f.objectGroupSize != 0 {
		panic(fmt.Sprintf("cache does not begin at mod: n=%d (mod=%d)", f.cacheS[0], f.cacheS[0] % f.objectGroupSize))
	}

	set := []AncientObjectS3{}
	uploads := []s3manager.BatchUploadObject{}
	remainders := []uint64{}
	for i, n := range f.cacheS {
		v := f.cache[n]
		set = append(set, v)
		remainders = append(remainders, n)

		// finalize upload object if we have the group-by number in the set, or if the item is the last
		endGroup := (n+1) % f.objectGroupSize == 0
		if endGroup || i == len(f.cacheS)-1 {
			// seal upload object
			b, err := json.Marshal(set)
			if err != nil {
				return err
			}
			set = []AncientObjectS3{}
			uploads = append(uploads, s3manager.BatchUploadObject{
				Object: &s3manager.UploadInput{
					Bucket: aws.String(f.bucketName()),
					Key:    aws.String(f.objectKeyForN(n)),
					Body:   bytes.NewReader(b),
				},
			})
		}
		if endGroup {
			remainders = remainders[:0]
		}
	}

	iter := &s3manager.UploadObjectsIterator{Objects: uploads}
	err := f.uploader.UploadWithIterator(aws.BackgroundContext(), iter)
	if err != nil {
		return err
	}
	// splice first n groups, leaving mod leftovers
	for _, n := range f.cacheS {
		if sliceIndexOf(remainders, n) < 0 {
			delete(f.cache, n)
		}
	}
	f.cacheS = remainders

	if f.cacheS[0] % f.objectGroupSize != 0 {
		panic(fmt.Sprintf("cache does not begin at mod: n=%d (mod=%d)", f.cacheS[0], f.cacheS[0] % f.objectGroupSize))
	}

	f.retrieved = map[uint64]AncientObjectS3{}

	return nil
}

// sync flushes all data tables to disk.
func (f *freezerRemoteS3) Sync() error {
	lenCache := len(f.cache)
	if lenCache == 0 {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	var err error

	f.log.Info("Syncing ancients", "blocks", lenCache)
	start := time.Now()

	err = f.pushCache()
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	blocksPerSecond := fmt.Sprintf("%0.2f", float64(lenCache)/elapsed.Seconds())

	err = f.setIndexMarker(atomic.LoadUint64(f.frozen))
	if err != nil {
		return err
	}

	f.log.Info("Finished syncing ancients", "blocks", lenCache, "elapsed", elapsed, "bps", blocksPerSecond)
	return err
}

// repair truncates all data tables to the same length.
func (f *freezerRemoteS3) repair() error {
	/*min := uint64(math.MaxUint64)
	for _, table := range f.tables {
		items := atomic.LoadUint64(&table.items)
		if min > items {
			min = items
		}
	}
	for _, table := range f.tables {
		if err := table.truncate(min); err != nil {
			return err
		}
	}
	atomic.StoreUint64(&f.frozen, min)
	*/
	return nil
}

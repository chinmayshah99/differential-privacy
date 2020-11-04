//
// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package pbeam

import (
	"fmt"

	log "github.com/golang/glog"
	"github.com/google/differential-privacy/go/checks"
	"github.com/google/differential-privacy/go/noise"
	"github.com/google/differential-privacy/privacy-on-beam/internal/kv"
	"github.com/apache/beam/sdks/go/pkg/beam"
	"github.com/apache/beam/sdks/go/pkg/beam/transforms/stats"
)

// CountParams specifies the parameters associated with a Count aggregation.
type CountParams struct {
	// Noise type (which is either LaplaceNoise{} or GaussianNoise{}).
	//
	// Defaults to LaplaceNoise{}.
	NoiseKind NoiseKind
	// Differential privacy budget consumed by this aggregation. If there is
	// only one aggregation, both Epsilon and Delta can be left 0; in that
	// case, the entire budget of the PrivacySpec is consumed.
	Epsilon, Delta float64
	// The maximum number of distinct values that a given privacy identifier
	// can influence. If a privacy identifier is associated with more values,
	// random values will be dropped. There is an inherent trade-off when
	// choosing this parameter: a larger MaxPartitionsContributed leads to less
	// data loss due to contribution bounding, but since the noise added in
	// aggregations is scaled according to maxPartitionsContributed, it also
	// means that more noise is added to each count.
	//
	// Required.
	MaxPartitionsContributed int64
	// The maximum number of times that a privacy identifier can contribute to
	// a single count (or, equivalently, the maximum value that a privacy
	// identifier can add to a single count in total). If MaxValue=10 and a
	// privacy identifier is associated with the same value in 15 records, Count
	// ignores 5 of these records and only adds 10 to the count for this value.
	// There is an inherent trade-off when choosing MaxValue: a larger
	// parameter means that less records are lost, but a larger noise.
	//
	// Required.
	MaxValue int64
	// You can input the list of partitions present in the output if you know
	// them in advance. When you specify partitions, partition selection /
	// thresholding will be disabled and partitions will appear in the output
	// if and only if they appear in the set of public partitions.
	//
	// You should not derive the list of partitions non-privately from private
	// data. You should only use this in either of the following cases:
	// 	1. The list of partitions is data-independent. For example, if you are
	// 	aggregating a metric by hour, you could provide a list of all possible
	// 	hourly period.
	// 	2. You use a differentially private operation to come up with the list of
	// 	partitions. For example, you could use the keys of a DistinctPrivacyID
	// 	operation as the list of public partitions.
	//
	// Note that current implementation limitations only allow up to millions of
	// public partitions.
	//
	// Optional.
	PublicPartitions beam.PCollection
}

// Count counts the number of times a value appears in a PrivatePCollection,
// adding differentially private noise to the counts and doing pre-aggregation
// thresholding to remove counts with a low number of distinct privacy
// identifiers. It is also possible to manually specify the list of partitions
// present in the output, in which case the partition selection/thresholding
// step is skipped.
//
// Note: Do not use when your results may cause overflows for int64 values.
// This aggregation is not hardened for such applications yet.
//
// Count transforms a PrivatePCollection<V> into a PCollection<V, int64>.
func Count(s beam.Scope, pcol PrivatePCollection, params CountParams) beam.PCollection {
	s = s.Scope("pbeam.Count")
	// Obtain type information from the underlying PCollection<K,V>.
	idT, partitionT := beam.ValidateKVType(pcol.col)

	// Get privacy parameters.
	spec := pcol.privacySpec
	epsilon, delta, err := spec.consumeBudget(params.Epsilon, params.Delta)
	if err != nil {
		log.Exitf("Couldn't consume budget: %v", err)
	}

	var noiseKind noise.Kind
	if params.NoiseKind == nil {
		noiseKind = noise.LaplaceNoise
		log.Infof("No NoiseKind specified, using Laplace Noise by default.")
	} else {
		noiseKind = params.NoiseKind.toNoiseKind()
	}
	err = checkCountParams(params, epsilon, delta, noiseKind)
	if err != nil {
		log.Exit(err)
	}

	maxPartitionsContributed := getMaxPartitionsContributed(spec, params.MaxPartitionsContributed)
	// Drop non-public partitions, if public partitions are specified.
	if (params.PublicPartitions).IsValid() {
		if partitionT.Type() != params.PublicPartitions.Type().Type() {
			log.Exitf("Public partitions must be of type %v. Got type %v instead.",
				partitionT.Type(), params.PublicPartitions.Type().Type())
		}
		partitionEncodedType := beam.EncodedType{partitionT.Type()}
		pcol.col = dropNonPublicPartitionsVFn(s, params.PublicPartitions, pcol, partitionEncodedType)
	}
	// First, encode KV pairs, count how many times each one appears,
	// and re-key by the original privacy key.
	coded := beam.ParDo(s, kv.NewEncodeFn(idT, partitionT), pcol.col)
	kvCounts := stats.Count(s, coded)
	counts64 := beam.ParDo(s, vToInt64Fn, kvCounts)
	rekeyed := beam.ParDo(s, rekeyInt64Fn, counts64)
	// Second, do cross-partition contribution bounding.
	rekeyed = boundContributions(s, rekeyed, maxPartitionsContributed)
	// Third, now that contribution bounding is done, remove the privacy keys,
	// decode the value, and sum all the counts bounded by maxCountContrib.
	countPairs := beam.DropKey(s, rekeyed)
	countsKV := beam.ParDo(s,
		newDecodePairInt64Fn(partitionT.Type()),
		countPairs,
		beam.TypeDefinition{Var: beam.XType, T: partitionT.Type()})
	// Add public partitions and return the aggregation output, if public partitions are specified.
	if (params.PublicPartitions).IsValid() {
		return addPublicPartitionsForCount(s, epsilon, delta, maxPartitionsContributed, params, noiseKind, countsKV)
	}
	sums := beam.CombinePerKey(s,
		newBoundedSumInt64Fn(epsilon, delta, maxPartitionsContributed, 0, params.MaxValue, noiseKind, false),
		countsKV)
	// Drop thresholded partitions.
	counts := beam.ParDo(s, dropThresholdedPartitionsInt64Fn, sums)
	// Clamp negative counts to zero and return.
	return beam.ParDo(s, clampNegativePartitionsInt64Fn, counts)
}

func checkCountParams(params CountParams, epsilon, delta float64, noiseKind noise.Kind) error {
	err := checks.CheckEpsilon("pbeam.Count", epsilon)
	if err != nil {
		return err
	}
	if (params.PublicPartitions).IsValid() && noiseKind == noise.LaplaceNoise {
		err = checks.CheckNoDelta("pbeam.Count", delta)
	} else {
		err = checks.CheckDeltaStrict("pbeam.Count", delta)
	}
	if err != nil {
		return err
	}
	err = checks.CheckMaxPartitionsContributed("pbeam.Count", params.MaxPartitionsContributed)
	if err != nil {
		return err
	}
	if params.MaxValue <= 0 {
		return fmt.Errorf("pbeam.Count: MaxValue should be strictly positive, got %d", params.MaxValue)
	}
	return nil
}

func addPublicPartitionsForCount(s beam.Scope, epsilon, delta float64, maxPartitionsContributed int64, params CountParams, noiseKind noise.Kind, countsKV beam.PCollection) beam.PCollection {
	// Turn PublicPartitions from PCollection<K> into PCollection<K, int64> by adding
	// the value zero to each K.
	dummyCounts := beam.ParDo(s, addDummyValuesToPublicPartitionsInt64Fn, params.PublicPartitions)
	// Merge countsKV and dummyCounts.
	allPartitions := beam.Flatten(s, dummyCounts, countsKV)
	// Sum and add noise.
	sums := beam.CombinePerKey(s, newBoundedSumInt64Fn(epsilon, delta, maxPartitionsContributed, 0, params.MaxValue, noiseKind, true), allPartitions)
	finalPartitions := beam.ParDo(s, dereferenceValueToInt64, sums)
	// Clamp negative counts to zero and return.
	return beam.ParDo(s, clampNegativePartitionsInt64Fn, finalPartitions)
}

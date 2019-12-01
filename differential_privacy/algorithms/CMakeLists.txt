# This file was generated from BUILD using tools/make_cmakelists.py.

cmake_minimum_required(VERSION 3.1)

if(${CMAKE_VERSION} VERSION_LESS 3.12)
    cmake_policy(VERSION ${CMAKE_MAJOR_VERSION}.${CMAKE_MINOR_VERSION})
else()
    cmake_policy(VERSION 3.12)
endif()

cmake_minimum_required (VERSION 3.0)
cmake_policy(SET CMP0048 NEW)



# Prevent CMake from setting -rdynamic on Linux (!!).
SET(CMAKE_SHARED_LIBRARY_LINK_C_FLAGS "")
SET(CMAKE_SHARED_LIBRARY_LINK_CXX_FLAGS "")

# Set default build type.
if(NOT CMAKE_BUILD_TYPE)
  message(STATUS "Setting build type to 'RelWithDebInfo' as none was specified.")
  set(CMAKE_BUILD_TYPE "RelWithDebInfo" CACHE STRING
      "Choose the type of build, options are: Debug Release RelWithDebInfo MinSizeRel."
      FORCE)
endif()

# When using Ninja, compiler output won't be colorized without this.
include(CheckCXXCompilerFlag)
CHECK_CXX_COMPILER_FLAG(-fdiagnostics-color=always SUPPORTS_COLOR_ALWAYS)
if(SUPPORTS_COLOR_ALWAYS)
  set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fdiagnostics-color=always")
endif()

# Implement ASAN/UBSAN options
if(UPB_ENABLE_ASAN)
  set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fsanitize=address")
  set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fsanitize=address")
  set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -fsanitize=address")
  set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} -fsanitize=address")
endif()

if(UPB_ENABLE_UBSAN)
  set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fsanitize=undefined")
  set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fsanitize=address")
  set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -fsanitize=address")
  set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} -fsanitize=address")
endif()

include_directories(.)
include_directories(${CMAKE_CURRENT_BINARY_DIR})

if(APPLE)
  set(CMAKE_SHARED_LINKER_FLAGS "${CMAKE_SHARED_LINKER_FLAGS} -undefined dynamic_lookup -flat_namespace")
elseif(UNIX)
  set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -Wl,--build-id")
endif()

find_package(Protobuf REQUIRED)
protobuf_generate_cpp(confidence_interval_cc_proto confidence-interval_proto confidence-interval.proto)

add_library(algorithm INTERFACE)
target_sources(algorithm INTERFACE algorithm.h)
target_link_libraries(algorithm INTERFACE 
  confidence_interval_cc_proto
  numerical-mechanisms
  util
  status
  statusor
  data_cc_proto
  summary_cc_proto
  util-lib
  memory)

add_library(binary-search INTERFACE)
target_sources(binary-search INTERFACE binary-search.h)
target_link_libraries(binary-search INTERFACE 
  algorithm
  numerical-mechanisms
  percentile
  status
  util-lib
  cc_wkt_protos)

add_library(bounded-algorithm INTERFACE)
target_sources(bounded-algorithm INTERFACE bounded-algorithm.h)
target_link_libraries(bounded-algorithm INTERFACE 
  algorithm
  approx-bounds
  status)

add_library(order-statistics INTERFACE)
target_sources(order-statistics INTERFACE order-statistics.h)
target_link_libraries(order-statistics INTERFACE 
  binary-search
  bounded-algorithm
  percentile
  status)

add_library(bounded-sum INTERFACE)
target_sources(bounded-sum INTERFACE bounded-sum.h)
target_link_libraries(bounded-sum INTERFACE 
  algorithm
  bounded-algorithm
  numerical-mechanisms
  status
  summary_cc_proto
  memory
  cc_wkt_protos)

add_library(bounded-mean INTERFACE)
target_sources(bounded-mean INTERFACE bounded-mean.h)
target_link_libraries(bounded-mean
  INTERFACE 
  algorithm
  bounded-algorithm
  numerical-mechanisms
  status
  summary_cc_proto
  distributions
  cc_wkt_protos)

add_library(bounded-variance INTERFACE)
target_sources(bounded-variance INTERFACE bounded-variance.h)
target_link_libraries(bounded-variance INTERFACE 
  algorithm
  approx-bounds
  bounded-algorithm
  numerical-mechanisms
  util-lib
  memory
  cc_wkt_protos)

add_library(bounded-standard-deviation INTERFACE)
target_sources(bounded-standard-deviation INTERFACE bounded-standard-deviation.h)
target_link_libraries(bounded-standard-deviation
  INTERFACE 
  algorithm
  bounded-algorithm
  bounded-variance
  numerical-mechanisms
  memory)

add_library(count INTERFACE)
target_sources(count INTERFACE count.h)
target_link_libraries(count
  INTERFACE 
  algorithm
  numerical-mechanisms
  summary_cc_proto
  cc_wkt_protos)

add_library(util STATIC
  util.cc
  util.h)
target_link_libraries(util
  logging
  status
  statusor
  strings)

add_library(distributions STATIC
  distributions.cc
  distributions.h)
target_link_libraries(distributions
  rand
  util
  statusor
  memory
  random
  strings)

add_library(numerical-mechanisms INTERFACE)
target_sources(numerical-mechanisms INTERFACE numerical-mechanisms.h)
target_link_libraries(numerical-mechanisms
  INTERFACE confidence_interval_cc_proto
  distributions
  util
  status)

# testing not needed
#add_library(numerical-mechanisms-testing INTERFACE)
#target_link_libraries(numerical-mechanisms-testing INTERFACE
#  confidence_interval_cc_proto
#  distributions
#  numerical-mechanisms
#  com_google_googletest//:gtest
#  /differential_privacy/base:statusor)

add_library(approx-bounds INTERFACE)
target_sources(approx-bounds INTERFACE approx-bounds.h)
target_link_libraries(approx-bounds
  INTERFACE algorithm
  status
  util-lib
  cc_wkt_protos)

add_library(rand STATIC
  rand.cc
  rand.h)
target_link_libraries(rand
  logging
  boringssl:crypto
  synchronization)


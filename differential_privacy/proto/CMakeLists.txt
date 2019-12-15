cmake_minimum_required(VERSION 3.3)
find_package(Protobuf REQUIRED)

SET(Protobuf_IMPORT_DIRS ../algorithms)
PROTOBUF_GENERATE_CPP(data_cc_proto data-proto data.proto)
add_library(data_cc_proto_lib ${data_cc_proto} ${data-proto})
target_link_libraries(data_cc_proto_lib confidence_interval_cc_proto_lib)

PROTOBUF_GENERATE_CPP(summary_cc_proto summary-proto summary.proto)
add_library(summary_cc_proto_lib ${summary_cc_proto} ${summary-proto})
target_link_libraries(summary_cc_proto_lib data_cc_proto_lib any_proto)

add_library(util-lib STATIC util.h)
target_link_libraries(util-lib 
data_cc_proto_lib 
confidence_interval_cc_proto_lib)

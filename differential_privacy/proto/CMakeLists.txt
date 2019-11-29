cmake_minimum_required(VERSION 3.3)
find_package(Protobuf REQUIRED)


PROTOBUF_GENERATE_CPP(data_cc_proto data-proto data.proto)
target_link_libraries(confidence-interval_proto)



PROTOBUF_GENERATE_CPP(summary_cc_proto summary-proto summary.proto)
target_link_libraries(data-proto 
any_proto) #TODO

add_library(util-lib STATIC util.h)
target_link_libraries(data_cc_proto)

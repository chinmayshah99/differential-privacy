cmake_minimum_required(VERSION 3.1)


#TODO
# proto_library cc_proto_library can you do this?

add_library(data-proto
data.proto)

add_library(data_cc_proto)
target_link_libraries(data-proto)


add_library(summary-proto summary.proto)
target_link_libraries(data-proto 
"@com_google_protobuf//:any_proto") #TODO

add_library(summary_cc_proto)
target_link_libraries(summary-proto)

add_library(util-lib util.h)
target_link_libraries(data_cc_proto)


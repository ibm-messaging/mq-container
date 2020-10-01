# © Copyright IBM Corporation 2017, 2020
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Where to find the sample test.
SRC_DIR = src

# Flags passed to the C compiler.
CFLAGS += -std=c11 -fPIC -Wall

log.o : $(SRC_DIR)/log.c $(SRC_DIR)/log.h
	gcc $(CFLAGS) -c $(SRC_DIR)/log.c

htpass.o : $(SRC_DIR)/htpass.c $(SRC_DIR)/htpass.h
	gcc $(CFLAGS) -c -I /usr/include/apr-1 $^

htpass_test.o : htpass.o log.o
	gcc $(CFLAGS) -I /usr/include/apr-1 -L/usr/lib64 -lapr-1 -laprutil-1 $(SRC_DIR)/htpass_test.c $^ -o htpass_test

htpass_test : htpass_test.o
	./htpass_test

mqhtpass.so : log.o htpass.o
	gcc $(CFLAGS) -I/opt/mqm/inc -D_REENTRANT -L/usr/lib64 -lapr-1 -laprutil-1 -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64 -shared $(SRC_DIR)/mqhtpass.c $^ -o $@
	ldd $@

mqhtpass_unittest.o : $(SRC_DIR)/mqhtpass_test.cc \
                     $(SRC_DIR)/sample1.h $(FUSED_GTEST_H)
	gcc $(CFLAGS) -c $(SRC_DIR)/mqhtpass_test.cc

mqhtpass_unittest : mqhtpass.so mqhtpass_test.o gtest-all.o gtest_main.o
	gcc $(CFLAGS) $^ -o $@
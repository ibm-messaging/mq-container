/*
© Copyright IBM Corporation 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package com.ibm.mqcontainer.test.framework;

/**
 * Assertion methods for writing tests.
 */
public class Assertions {
    
    /**
     * Assert that the given object is not null.
     */
    public static void assertNotNull(Object object) {
        assertNotNull(object, "Expected non-null value");
    }

    /**
     * Assert that the given object is not null.
     */
    public static void assertNotNull(Object object, String message) {
        if (object == null) {
            throw new AssertionError(message);
        }
    }

    /**
     * Assert that the condition is true.
     */
    public static void assertTrue(boolean condition) {
        assertTrue(condition, "Expected condition to be true");
    }

    /**
     * Assert that the condition is true.
     */
    public static void assertTrue(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    /**
     * Assert that the condition is false.
     */
    public static void assertFalse(boolean condition) {
        assertFalse(condition, "Expected condition to be false");
    }

    /**
     * Assert that the condition is false.
     */
    public static void assertFalse(boolean condition, String message) {
        if (condition) {
            throw new AssertionError(message);
        }
    }

    /**
     * Assert that two objects are equal.
     */
    public static void assertEquals(Object expected, Object actual) {
        assertEquals(expected, actual, "Expected values to be equal");
    }

    /**
     * Assert that two objects are equal.
     */
    public static void assertEquals(Object expected, Object actual, String message) {
        if (expected == null && actual == null) {
            return;
        }
        if (expected == null || actual == null) {
            throw new AssertionError(message + " - expected: " + expected + ", actual: " + actual);
        }
        if (!expected.equals(actual)) {
            throw new AssertionError(message + " - expected: " + expected + ", actual: " + actual);
        }
    }

    /**
     * Assert that two integers are equal.
     */
    public static void assertEquals(int expected, int actual) {
        assertEquals(expected, actual, "Expected values to be equal");
    }

    /**
     * Assert that two integers are equal.
     */
    public static void assertEquals(int expected, int actual, String message) {
        if (expected != actual) {
            throw new AssertionError(message + " - expected: " + expected + ", actual: " + actual);
        }
    }

    /**
     * Fail the test with the given message.
     */
    public static void fail(String message) {
        throw new AssertionError(message);
    }

    /**
     * Fail the test.
     */
    public static void fail() {
        throw new AssertionError("Test failed");
    }
}
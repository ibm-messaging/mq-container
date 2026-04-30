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

import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.util.ArrayList;
import java.util.List;

/**
 * Simple test runner that discovers and executes tests.
 */
public class TestRunner {
    
    private int totalTests = 0;
    private int passedTests = 0;
    private int failedTests = 0;
    private int skippedTests = 0;

    /**
     * Run all tests in the given class.
     */
    public void runTestClass(Class<?> testClass) throws Exception {
        System.out.println("\n========================================");
        System.out.println("Running tests in: " + testClass.getName());
        System.out.println("========================================\n");

        // Find all methods with annotations
        List<Method> beforeAllMethods = new ArrayList<>();
        List<Method> beforeEachMethods = new ArrayList<>();
        List<Method> testMethods = new ArrayList<>();
        List<Method> afterEachMethods = new ArrayList<>();
        List<Method> afterAllMethods = new ArrayList<>();

        for (Method method : testClass.getDeclaredMethods()) {
            if (method.isAnnotationPresent(BeforeAll.class)) {
                beforeAllMethods.add(method);
            }
            if (method.isAnnotationPresent(BeforeEach.class)) {
                beforeEachMethods.add(method);
            }
            if (method.isAnnotationPresent(Test.class)) {
                testMethods.add(method);
            }
            if (method.isAnnotationPresent(AfterEach.class)) {
                afterEachMethods.add(method);
            }
            if (method.isAnnotationPresent(AfterAll.class)) {
                afterAllMethods.add(method);
            }
        }

        // Run @BeforeAll methods
        for (Method method : beforeAllMethods) {
            method.setAccessible(true);
            if (!Modifier.isStatic(method.getModifiers())) {
                throw new RuntimeException("@BeforeAll method must be static: " + method.getName());
            }
            method.invoke(null);
        }

        // Run each test
        for (Method testMethod : testMethods) {
            totalTests++;
            
            // Check if test is disabled
            if (testMethod.isAnnotationPresent(Disabled.class)) {
                skippedTests++;
                System.out.println("⊘ SKIPPED: " + testMethod.getName());
                Disabled disabled = testMethod.getAnnotation(Disabled.class);
                if (!disabled.value().isEmpty()) {
                    System.out.println("  Reason: " + disabled.value());
                }
                continue;
            }

            Object testInstance = testClass.getDeclaredConstructor().newInstance();
            
            try {
                // Run @BeforeEach methods
                for (Method method : beforeEachMethods) {
                    method.setAccessible(true);
                    method.invoke(testInstance);
                }

                // Run the test
                testMethod.setAccessible(true);
                
                // Check if test method takes TestInfo parameter
                Class<?>[] paramTypes = testMethod.getParameterTypes();
                if (paramTypes.length == 1 && paramTypes[0].equals(TestInfo.class)) {
                    TestInfo testInfo = new TestInfo(testMethod.getName());
                    testMethod.invoke(testInstance, testInfo);
                } else if (paramTypes.length == 0) {
                    testMethod.invoke(testInstance);
                } else {
                    throw new RuntimeException("Test method has unsupported parameters: " + testMethod.getName());
                }

                passedTests++;
                System.out.println("✓ PASSED: " + testMethod.getName());

            } catch (Exception e) {
                failedTests++;
                System.out.println("✗ FAILED: " + testMethod.getName());
                
                Throwable cause = e.getCause();
                if (cause != null) {
                    System.out.println("  Error: " + cause.getClass().getSimpleName() + ": " + cause.getMessage());
                    cause.printStackTrace(System.out);
                } else {
                    System.out.println("  Error: " + e.getMessage());
                    e.printStackTrace(System.out);
                }
            } finally {
                // Run @AfterEach methods
                for (Method method : afterEachMethods) {
                    try {
                        method.setAccessible(true);
                        method.invoke(testInstance);
                    } catch (Exception e) {
                        System.out.println("  Warning: @AfterEach method failed: " + e.getMessage());
                    }
                }
            }
        }

        // Run @AfterAll methods
        for (Method method : afterAllMethods) {
            try {
                method.setAccessible(true);
                if (!Modifier.isStatic(method.getModifiers())) {
                    throw new RuntimeException("@AfterAll method must be static: " + method.getName());
                }
                method.invoke(null);
            } catch (Exception e) {
                System.out.println("Warning: @AfterAll method failed: " + e.getMessage());
            }
        }
    }

    /**
     * Print test summary.
     */
    public void printSummary() {
        System.out.println("\n========================================");
        System.out.println("Test Summary");
        System.out.println("========================================");
        System.out.println("Total:   " + totalTests);
        System.out.println("Passed:  " + passedTests);
        System.out.println("Failed:  " + failedTests);
        System.out.println("Skipped: " + skippedTests);
        System.out.println("========================================\n");
    }

    /**
     * Check if all tests passed.
     */
    public boolean allTestsPassed() {
        return failedTests == 0 && totalTests > 0;
    }

    /**
     * Main entry point for running tests.
     */
    public static void main(String[] args) {
        if (args.length == 0) {
            System.err.println("Usage: java TestRunner <test-class-name> [<test-class-name> ...]");
            System.exit(1);
        }

        TestRunner runner = new TestRunner();
        boolean allPassed = true;

        for (String className : args) {
            try {
                Class<?> testClass = Class.forName(className);
                runner.runTestClass(testClass);
            } catch (ClassNotFoundException e) {
                System.err.println("Error: Test class not found: " + className);
                allPassed = false;
            } catch (Exception e) {
                System.err.println("Error running tests in " + className + ": " + e.getMessage());
                e.printStackTrace();
                allPassed = false;
            }
        }

        runner.printSummary();

        if (!runner.allTestsPassed() || !allPassed) {
            System.exit(1);
        }
    }
}
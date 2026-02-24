# Task: Add Regression Test Suite for UI Task Tree

## Context
The UI task tree hierarchy (Root -> Task -> Run) is the central visualization of the workflow. Recent regressions have caused incorrect rendering or misaligned nodes, making it difficult to understand the execution flow.

## Goal
Establish a regression test suite specifically for the UI task tree rendering logic to prevent future regressions.

## Objectives
1.  **Identify Test Points**: Determine the critical rendering logic for the task tree, including:
    -   Root task display.
    -   Individual task nodes and their hierarchy.
    -   Execution run details associated with tasks.
    -   Parent-child relationships and nesting.
2.  **Implement Tests**: Create a comprehensive test suite (e.g., using Jest, Cypress, or equivalent available in the project) that verifies:
    -   Correct rendering of the tree structure for known data.
    -   Correct handling of empty states or single-node trees.
    -   Correct update behavior when new nodes are added.
3.  **Documentation**: Add instructions on how to run these specific tests.

## Verification
1.  **Execute Tests**: Run the new test suite and ensure all tests pass.
2.  **Negative Test**: Temporarily modify the tree rendering logic to introduce a bug (e.g., incorrect parent ID lookup) and verify that the tests fail.
3.  **Coverage**: Ensure the tests cover the core scenarios of task hierarchy visualization.

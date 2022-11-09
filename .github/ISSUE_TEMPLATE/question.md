---
name: Question
about: Title of your question.
title: ''
labels: question
assignees: ''
---

Rod Version: v0.0.0

## The code to demonstrate your question

1. Clone Rod to your local and cd to the repository:

   ```bash
   git clone https://github.com/go-rod/rod
   cd rod
   ```

1. Use your code to replace the content of function `TestRod` in file `rod_test.go`.

1. Test your code with: `go test -run TestRod`, make sure it fails as expected.

1. Replace ALL THE CONTENT under "The code to demonstrate your question" with your `TestRod` function, like below:

```go
func TestRod(t *testing.T) {
    g := setup(t)
    g.Eq(1, 2) // the test should fail, here 1 doesn't equal 2
}
```

## What you got

Such as what error you see.

## What you expected to see

Such as what you want to do.

## What have you tried to solve the question

Such as after modifying some source code of Rod you are able to get rid of the problem.

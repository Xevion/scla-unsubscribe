package main

// A channel that will be used to buffer incomplete entries that need to be queried properly
var incompleteEntries = make(chan Entry)

// A channel that will be used to buffer emails that need to be unsubscribed
var entries = make(chan string)

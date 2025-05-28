//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/playwright-community/playwright-go"
)

func must(message string, err error) {
	if err != nil {
		log.Fatalf(message, err)
	}
}

func eq(expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		panic(fmt.Sprintf("%v does not equal %v", actual, expected))
	}
}

const taskName = "Bake a cake"
const initialURL = "https://demo.playwright.dev/todomvc/"

func main() {
	pw, err := playwright.Run()
	must("could not launch playwright: %w", err)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	must("could not launch Chromium: %w", err)
	context, err := browser.NewContext()
	must("could not create context: %w", err)
	page, err := context.NewPage()
	must("could not create page: %w", err)
	_, err = page.Goto(initialURL)
	must("could not goto: %w", err)

	// Helper function to get the amount of todos on the page
	shouldTaskCount := func(shouldBeCount int) {
		count, err := page.Locator("ul.todo-list > li").Count()
		must("could not determine todo list count: %w", err)
		eq(shouldBeCount, count)
	}

	// Initially there should be 0 entries
	shouldTaskCount(0)

	newTodoInput := page.Locator("input.new-todo")
	// Adding a todo entry (click in the input, enter the todo title and press the Enter key)
	must("could not click: %v", newTodoInput.Click())
	must("could not type: %v", newTodoInput.Fill(taskName))
	must("could not press: %v", newTodoInput.Press("Enter"))

	// After adding 1 there should be 1 entry in the list
	shouldTaskCount(1)

	// Here we get the text in the first todo item to see if it"s the same which the user has entered
	textContentOfFirstTodoEntry, err := page.Locator("ul.todo-list > li:nth-child(1) label").Evaluate("el => el.textContent", nil)
	must("could not get first todo entry: %w", err)
	eq(taskName, textContentOfFirstTodoEntry)

	// The todo list should be persistent. Here we reload the page and see if the entry is still there
	_, err = page.Reload()
	must("could not reload: %w", err)
	shouldTaskCount(1)

	// Set the entry to completed
	must("could not click: %v", page.Locator("input.toggle").Click())

	// Filter for active entries. There should be 0, because we have completed the entry already
	must("could not click: %v", page.Locator("text=Active").Click())
	shouldTaskCount(0)

	// If we filter now for completed entries, there should be 1
	must("could not click: %v", page.GetByRole("link", playwright.PageGetByRoleOptions{
		Name: "Completed",
	}).Click())
	shouldTaskCount(1)

	// Clear the list of completed entries, then it should be again 0
	must("could not click: %v", page.Locator("text=Clear completed").Click())
	shouldTaskCount(0)

	must("could not close browser: %w", browser.Close())
	must("could not stop Playwright: %w", pw.Stop())
}

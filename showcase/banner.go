func printBanner() {
	fmt.Println()
	fmt.Printf("%s%s", bold, cyan)

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║                 LAT                  ║")
	fmt.Println("║                 :3                   ║")
	fmt.Println("╚══════════════════════════════════════╝")

	fmt.Printf("%s", reset)
	fmt.Println()
}

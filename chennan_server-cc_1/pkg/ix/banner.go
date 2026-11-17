package ix

import "fmt"

/*
** Slant GolangAppKit
** http://patorjk.com/software/taag
 */
func ShowBanner() {
	fmt.Printf(`>>> %s ---> %s
    _ _  __                __      ___                __ __ _ __ 
   (_) |/ /___ ___  ______/ /___ _/   |  ____  ____  / //_/(_) /_
  / /|   / __  / / / / __  / __  / /| | / __ \/ __ \/ ,<  / / __/
 / //   / /_/ / /_/ / /_/ / /_/ / ___ |/ /_/ / /_/ / /| |/ / /_  
/_//_/|_\__,_/\__, /\__,_/\__,_/_/  |_/ .___/ .___/_/ |_/_/\__/  
             /____/                  /_/   /_/ %s
`, Services, MicroService, Version.ToString())
}

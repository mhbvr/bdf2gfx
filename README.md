# bdf2gfx

Converter fom bdf pixel font format to GFX structure that uses by AdafuitGFX and TFT_eSPI libs.

The code is originally written by LLM (Deepseek), but requred some fixes to work properly:

* Bitmap parsing moved to the main loop.
* Cleanup of the final output.

This is one time script, so it is not handling corner cases and produce not optimize bitmap data. But the result is usable with TFT_eSPI. One can also check the .h file with online GFX editor: 
https://tchapi.github.io/Adafruit-GFX-Font-Customiser/
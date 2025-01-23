# My setup: 

raspberry pi: 
* openwakeword https://github.com/rhasspy/wyoming-openwakeword/
* https://github.com/rhasspy/wyoming-satellite
* goyoming event listener

homeassistant:
* chime_tts
* whisper + piper tts
* sonos media players (alexa doesn't work here since it can't play TTS directly without announce modes)

# What does goyoming do? 
* Detect wake word events and plays a notification sound from HA media library on target speaker
* Detect voice stop events and plays a notification sound from HA media library on target speaker
* Detect synthesize events, cleans the payload and sends it to HA chime_tts service on target speaker with selected platform and voice
* Reduces TTS volume level by 15% during "quite" hours (hard coded 8pm -> 8am of device time)
* Monitors event flow for stuck satellite and issues a restart command (expected satellite running as a systemd unit called 'satellite.service')

# What does it NOT do?
* Handle faster_whisper engine crashes / disconnects (not running on raspberry)
* Handle network disconnect/failures very well (satellite disconnects and doesn't respond to wake words)
* Is flexible enough to cover arbitrary setups 

There's more generic stuff I could add here but it's for me :D 


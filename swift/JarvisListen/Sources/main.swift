import Foundation
import Speech
import AVFoundation

/// JarvisListen - Speech-to-text CLI using macOS SFSpeechRecognizer
/// Usage: JarvisListen [--stream] [--timeout SECONDS]
///   --stream: Output partial results as they become available
///   --timeout: Maximum recording time in seconds (default: 30)

class SpeechRecognizer {
    private let speechRecognizer: SFSpeechRecognizer
    private let audioEngine: AVAudioEngine
    private var recognitionRequest: SFSpeechAudioBufferRecognitionRequest?
    private var recognitionTask: SFSpeechRecognitionTask?
    private var isStreaming: Bool
    private var timeout: TimeInterval
    private var lastTranscription: String = ""
    private var silenceTimer: Timer?
    private let silenceThreshold: TimeInterval = 2.0 // Stop after 2 seconds of silence

    init(streaming: Bool, timeout: TimeInterval) {
        guard let recognizer = SFSpeechRecognizer(locale: Locale(identifier: "en-US")) else {
            fputs("Speech recognition not available\n", stderr)
            exit(1)
        }

        self.speechRecognizer = recognizer
        self.audioEngine = AVAudioEngine()
        self.isStreaming = streaming
        self.timeout = timeout
    }

    func requestAuthorization() async -> Bool {
        await withCheckedContinuation { continuation in
            SFSpeechRecognizer.requestAuthorization { status in
                switch status {
                case .authorized:
                    continuation.resume(returning: true)
                case .denied:
                    fputs("Speech recognition access denied\n", stderr)
                    continuation.resume(returning: false)
                case .restricted:
                    fputs("Speech recognition restricted on this device\n", stderr)
                    continuation.resume(returning: false)
                case .notDetermined:
                    fputs("Speech recognition not determined\n", stderr)
                    continuation.resume(returning: false)
                @unknown default:
                    fputs("Unknown speech recognition status\n", stderr)
                    continuation.resume(returning: false)
                }
            }
        }
    }

    func startRecognition() async throws -> String {
        guard speechRecognizer.isAvailable else {
            throw NSError(domain: "JarvisListen", code: 1,
                          userInfo: [NSLocalizedDescriptionKey: "Speech recognizer not available"])
        }

        // Create recognition request
        recognitionRequest = SFSpeechAudioBufferRecognitionRequest()
        guard let recognitionRequest = recognitionRequest else {
            throw NSError(domain: "JarvisListen", code: 2,
                          userInfo: [NSLocalizedDescriptionKey: "Unable to create recognition request"])
        }

        recognitionRequest.shouldReportPartialResults = true
        recognitionRequest.addsPunctuation = true

        // Get audio input
        let inputNode = audioEngine.inputNode
        let recordingFormat = inputNode.outputFormat(forBus: 0)

        inputNode.installTap(onBus: 0, bufferSize: 1024, format: recordingFormat) { buffer, _ in
            self.recognitionRequest?.append(buffer)
            self.resetSilenceTimer()
        }

        audioEngine.prepare()
        try audioEngine.start()

        // Print listening indicator to stderr
        fputs("Listening...\n", stderr)

        return try await withCheckedThrowingContinuation { continuation in
            var hasResumed = false

            recognitionTask = speechRecognizer.recognitionTask(with: recognitionRequest) { [weak self] result, error in
                guard let self = self else { return }

                if let result = result {
                    let transcription = result.bestTranscription.formattedString
                    self.lastTranscription = transcription

                    if self.isStreaming {
                        // Output partial results immediately
                        print(transcription)
                        fflush(stdout)
                    }

                    if result.isFinal {
                        self.stopRecognition()
                        if !hasResumed {
                            hasResumed = true
                            continuation.resume(returning: transcription)
                        }
                    }
                }

                if let error = error {
                    self.stopRecognition()
                    if !hasResumed {
                        hasResumed = true
                        // If we have some transcription, return it despite the error
                        if !self.lastTranscription.isEmpty {
                            continuation.resume(returning: self.lastTranscription)
                        } else {
                            continuation.resume(throwing: error)
                        }
                    }
                }
            }

            // Set up timeout
            DispatchQueue.main.asyncAfter(deadline: .now() + self.timeout) { [weak self] in
                guard let self = self else { return }
                if !hasResumed {
                    hasResumed = true
                    self.stopRecognition()
                    continuation.resume(returning: self.lastTranscription)
                }
            }
        }
    }

    private func resetSilenceTimer() {
        silenceTimer?.invalidate()
        silenceTimer = Timer.scheduledTimer(withTimeInterval: silenceThreshold, repeats: false) { [weak self] _ in
            self?.recognitionRequest?.endAudio()
        }
    }

    func stopRecognition() {
        silenceTimer?.invalidate()
        silenceTimer = nil

        audioEngine.stop()
        audioEngine.inputNode.removeTap(onBus: 0)

        recognitionRequest?.endAudio()
        recognitionRequest = nil

        recognitionTask?.cancel()
        recognitionTask = nil
    }
}

// Parse command line arguments
var streaming = false
var timeout: TimeInterval = 30

var args = CommandLine.arguments.dropFirst()
while let arg = args.first {
    args = args.dropFirst()
    switch arg {
    case "--stream":
        streaming = true
    case "--timeout":
        if let nextArg = args.first, let value = TimeInterval(nextArg) {
            timeout = value
            args = args.dropFirst()
        }
    case "--help", "-h":
        print("""
        JarvisListen - Speech-to-text using macOS SFSpeechRecognizer

        Usage: JarvisListen [options]

        Options:
          --stream          Output partial results as they become available
          --timeout SECS    Maximum recording time in seconds (default: 30)
          --help, -h        Show this help message
        """)
        exit(0)
    default:
        fputs("Unknown argument: \(arg)\n", stderr)
        exit(1)
    }
}

// Run the recognizer
let recognizer = SpeechRecognizer(streaming: streaming, timeout: timeout)

Task {
    let authorized = await recognizer.requestAuthorization()
    guard authorized else {
        exit(1)
    }

    do {
        let transcription = try await recognizer.startRecognition()
        if !streaming {
            // Only print final result if not streaming (streaming prints partial results)
            print(transcription)
        }
        exit(0)
    } catch {
        fputs("Error: \(error.localizedDescription)\n", stderr)
        exit(1)
    }
}

// Keep the main thread alive
RunLoop.main.run()

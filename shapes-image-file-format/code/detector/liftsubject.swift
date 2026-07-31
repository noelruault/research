// liftsubject — Apple's Vision foreground-instance mask (the engine behind "Lift Subject"),
// as a CLI, so this study can compare against the incumbent instead of speculating about it.
//
// Zero dependencies: Vision ships with macOS. Verified available on macOS 26.5.2 / Swift 6.3.3
// before this file was written, rather than assumed from memory.
//
// usage: liftsubject <in.png|jpg> <out-mask.png>
// Writes an 8-bit greyscale PNG: 255 = foreground (subject), 0 = background.

import Foundation
import Vision
import CoreImage
import AppKit

guard CommandLine.arguments.count >= 3 else {
    FileHandle.standardError.write("usage: liftsubject <in.png> <out-mask.png>\n".data(using: .utf8)!)
    exit(2)
}
let inPath = CommandLine.arguments[1]
let outPath = CommandLine.arguments[2]

guard let src = CIImage(contentsOf: URL(fileURLWithPath: inPath)) else {
    FileHandle.standardError.write("cannot read \(inPath)\n".data(using: .utf8)!)
    exit(1)
}

let handler = VNImageRequestHandler(ciImage: src, options: [:])
let request = VNGenerateForegroundInstanceMaskRequest()

do {
    try handler.perform([request])
} catch {
    FileHandle.standardError.write("vision failed: \(error)\n".data(using: .utf8)!)
    exit(1)
}

guard let obs = request.results?.first else {
    FileHandle.standardError.write("no foreground instance found\n".data(using: .utf8)!)
    exit(3)
}

// allInstances excludes the background; union them so a multi-part subject stays whole.
let instances = obs.allInstances
// VNObservation exposes a confidence: how much to trust the mask, not just what the mask is.
// The SHPC selection chunk stores it (mode 2) so a consumer can tell an uncertain cut from a
// confident one without re-running anything.
FileHandle.standardError.write("instances: \(instances.count)\n".data(using: .utf8)!)
FileHandle.standardError.write("confidence: \(obs.confidence)\n".data(using: .utf8)!)
// Third argument: write the confidence to a file so the encoder can store it.
if CommandLine.arguments.count >= 4 {
    try? "\(obs.confidence)".write(toFile: CommandLine.arguments[3], atomically: true, encoding: .utf8)
}

let pixelBuffer: CVPixelBuffer
do {
    pixelBuffer = try obs.generateScaledMaskForImage(forInstances: instances, from: handler)
} catch {
    FileHandle.standardError.write("mask generation failed: \(error)\n".data(using: .utf8)!)
    exit(1)
}

// The mask comes back as a one-channel float buffer; render it to 8-bit grey via CoreImage.
let maskCI = CIImage(cvPixelBuffer: pixelBuffer)
let ctx = CIContext()
guard let cg = ctx.createCGImage(maskCI, from: maskCI.extent) else {
    FileHandle.standardError.write("cannot rasterise mask\n".data(using: .utf8)!)
    exit(1)
}
let rep = NSBitmapImageRep(cgImage: cg)
guard let png = rep.representation(using: .png, properties: [:]) else {
    FileHandle.standardError.write("cannot encode png\n".data(using: .utf8)!)
    exit(1)
}
try png.write(to: URL(fileURLWithPath: outPath))
FileHandle.standardError.write("wrote \(outPath) (\(cg.width)x\(cg.height))\n".data(using: .utf8)!)

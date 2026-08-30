#!/usr/bin/env ruby
# encoding: utf-8

RED = "\033[0;31m"
GRN = "\033[0;32m"
NC = "\033[0m"

root = File.expand_path(__dir__)
require "socket"
require "open3"
require "timeout"
binary_names = Gem.win_platform? ? %w[gocode.exe gocode] : %w[gocode]
binary = binary_names.map { |name| File.join(root, name) }.find { |path| File.file?(path) }
abort "gocode binary not found in #{root}" unless binary
port = 40000 + (Process.pid % 1000)
addr = "127.0.0.1:#{port}"
server = Process.spawn(binary, "-s", "-sock", "tcp", "-addr", addr,
                       chdir: root, out: File::NULL, err: File::NULL)
at_exit do
  Open3.capture2(binary, "-sock", "tcp", "-addr", addr, "close") rescue nil
  if Gem.win_platform?
    system("taskkill", "/PID", server.to_s, "/T", "/F", out: File::NULL, err: File::NULL)
  else
    Process.kill("TERM", server) rescue nil
  end
  Timeout.timeout(5) { Process.wait(server) }
rescue Timeout::Error
end

100.times do
  begin
    TCPSocket.new("127.0.0.1", port).close
    break
  rescue Errno::ECONNREFUSED
    sleep 0.01
  end
end

tests = ARGV.empty? ? Dir["#{root}/typesinfo.*"].sort : ARGV.map { |t| File.expand_path(t, root) }
total = ok = 0
tests.each do |dir|
  next unless File.directory?(dir)
  total += 1
  file = File.join(dir, "test.go.in")
  cursor = Dir[File.join(dir, "cursor.*")].first.split(".").last.to_i
  source = File.binread(file)
  cursor += source.byteslice(0, cursor).count("\n") if source.include?("\r\n")
  expected = File.read(File.join(dir, "out.expected"))
  out, status = Open3.capture2(binary, "-sock", "tcp", "-addr", addr,
                               "-in", file, "liteide_typesinfo", file, cursor.to_s)
  out = "" unless status.success?
  out = out.tr("\\", "/")
  out = out.gsub(File.expand_path(file).tr("\\", "/"), "FILE")
  goroot = ENV["GOROOT"] || `go env GOROOT`.strip
  out = out.gsub(goroot, "GOROOT")
  out = out.gsub(%r{.*/go/(?:\d+\.)+\d+/(?:[^/]+/)?src/}, "GOROOT/src/")
  out = out.gsub(%r{.*/go\d+(?:\.\d+)+/(?:[^/]+/)?src/}, "GOROOT/src/")
  out = out.gsub(%r{GOROOT/src/([^:]+):\d+:\d+}, 'GOROOT/src/\\1:LINE:COLUMN')
  if out == expected
    puts "#{File.basename(dir)}: #{GRN}PASS!#{NC}"
    ok += 1
  else
    puts "#{File.basename(dir)}: #{RED}FAIL!#{NC}"
    puts "Got:\n#{out}Expected:\n#{expected}"
  end
end
puts "\nSummary (total: #{total})"
puts "#{GRN}  PASS#{NC}: #{ok}"
puts "#{RED}  FAIL#{NC}: #{total - ok}"
exit(total == ok ? 0 : 1)

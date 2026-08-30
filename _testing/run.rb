#!/usr/bin/env ruby
# encoding: utf-8

RED = "\033[0;31m"
GRN = "\033[0;32m"
NC  = "\033[0m"

PASS = "#{GRN}PASS!#{NC}"
FAIL = "#{RED}FAIL!#{NC}"

Stats = Struct.new :total, :ok, :fail
$stats = Stats.new 0, 0, 0
require "open3"
root = File.expand_path(__dir__)
names = Gem.win_platform? ? %w[gocode.exe gocode] : %w[gocode]
$gocode = names.map { |name| File.join(root, name) }.find { |path| File.file?(path) }
abort "gocode binary not found in #{root}" unless $gocode

def print_fail_report(t, out, outexpected)
	puts "#{t}: #{FAIL}"
	puts "-"*65
	puts "Got:\n#{out}"
	puts "-"*65
	puts "Expected:\n#{outexpected}"
	puts "-"*65
end

def print_pass_report(t)
	puts "#{t}: #{PASS}"
end

def print_stats
	puts "\nSummary (total: #{$stats.total})"
	puts "#{GRN}  PASS#{NC}: #{$stats.ok}"
	puts "#{RED}  FAIL#{NC}: #{$stats.fail}"
	puts "#{$stats.fail == 0 ? GRN : RED}#{"█"*72}#{NC}"
end

def run_test(t)
	$stats.total += 1

	cursorpos = Dir["#{t}/cursor.*"].map{|d| File.extname(d)[1..-1]}.first
	version = `go env GOVERSION 2>/dev/null`.strip.sub(/^go/, '')
	major_minor = version.split('.')[0, 2].join('.')
	windows = ENV["RUNNER_OS"] == "Windows" || File::ALT_SEPARATOR == "\\" || ENV["OS"] == "Windows_NT" || Gem.win_platform? || RUBY_PLATFORM =~ /mingw|mswin|msys/
	platform = windows ? ".windows" : ""
	versioned = "#{t}/out.expected.go#{major_minor}#{platform}"
	versioned = "#{t}/out.expected.go#{major_minor}" unless File.file?(versioned)
	expected_file = File.file?(versioned) ? versioned : "#{t}/out.expected"
	outexpected = IO.read(expected_file) rescue "To be determined"
	filename = "#{t}/test.go.in"

	out, status = Open3.capture2($gocode, "-in", filename, "autocomplete", filename, cursorpos.to_s)
	out = "" unless status.success?

	if out != outexpected then
		print_fail_report(t, out, outexpected)
		$stats.fail += 1
	else
		print_pass_report(t)
		$stats.ok += 1
	end
end

if ARGV.one?
	run_test ARGV[0]
else
	Dir["test.*"].sort.each do |t| 
		run_test t
	end
end

print_stats
exit($stats.fail == 0 ? 0 : 1)

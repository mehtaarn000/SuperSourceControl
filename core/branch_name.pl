# Regex statement copied from stackoverflow
my $valid_ref_name = qr%
   ^
   (?!
      # begins with
      /|                # (from #6)   cannot begin with /
      # contains
      .*(?:
         [/.]\.|        # (from #1,3) cannot contain /. or ..
         //|            # (from #6)   cannot contain multiple consecutive slashes
         @\{|           # (from #8)   cannot contain a sequence @{
         \\             # (from #9)   cannot contain a \
      )
   )
                        # (from #2)   (waiving this rule; too strict)
   [^\040\177 ~^:?*[]+  # (from #4-5) valid character rules

   # ends with
   (?<!\.lock)          # (from #1)   cannot end with .lock
   (?<![/.])            # (from #6-7) cannot end with / or .
   $
%x;

# if valid branch name print "true", else "false"
my $branch = @ARGV[0];
if ($branch =~ $valid_ref_name) {
    print "true";
} else {
    print "false";
}
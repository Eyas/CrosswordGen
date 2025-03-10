//using static WordGenLib.Generator;
using Crossword;
using System.Collections.Immutable;

namespace WordGenLib
{

    internal record struct ConcreteLine(string Line, ImmutableArray<string> Words) { }
    internal record struct ChoiceStep(IPossibleLines Choice, IPossibleLines Remaining) { }

    internal interface IPossibleLines
    {
        int NumLetters { get; }
        long MaxPossibilities { get; }
        void CharsAt(CharSet accumulate, int index);
        bool DefinitelyBlockedAt(int index);
        IPossibleLines Filter(CharSet constraint, int index);
        IPossibleLines Filter(char constraint, int index);
        IPossibleLines RemoveWordOption(string word);
        IEnumerable<ConcreteLine> Iterate();
        ConcreteLine? FirstOrNull();
        ChoiceStep MakeChoice();

        public static IPossibleLines RemoveWordOptions(IEnumerable<string> line, IPossibleLines l)
        {
            foreach (string w in line)
            {
                if (string.IsNullOrEmpty(w)) continue;
                l = l.RemoveWordOption(w);
            }
            return l;
        }
    }
    record Impossible(int NumLetters) : IPossibleLines
    {
        public long MaxPossibilities => 0;
        public void CharsAt(CharSet accumulate, int _) { }
        public bool DefinitelyBlockedAt(int _) => false;
        public IPossibleLines Filter(CharSet _, int _2) => this;
        public IPossibleLines Filter(char _, int _2) => this;
        public IPossibleLines RemoveWordOption(string word) => this;
        public IEnumerable<ConcreteLine> Iterate() => [];
        public ConcreteLine? FirstOrNull() => null;
        public ChoiceStep MakeChoice() => throw new InvalidOperationException("Cannot call MakeChoice on Impossible");

        private static readonly Impossible?[] _cache = new Impossible?[25];
        public static Impossible Instance(int numLetters)
        {
            if (_cache[numLetters] == null) _cache[numLetters] = new Impossible(numLetters);

            return _cache[numLetters]!;
        }
    }
    record Words(ImmutableArraySegment<string> Preferred, ImmutableArraySegment<string> Obscure) : IPossibleLines
    {
        public int NumLetters => (Preferred.Length > 0 ? Preferred[0].Length : Obscure[0].Length);
        public long MaxPossibilities => Preferred.Length + Obscure.Length;
        public void CharsAt(CharSet accumulate, int index)
        {
            if (accumulate.IsFull) return;

            foreach (var word in Preferred.Concat(Obscure))
            {
                accumulate.Add(word[index]);
                if (accumulate.IsFull) return;
            }
        }
        public bool DefinitelyBlockedAt(int index)
            => Preferred.Concat(Obscure).All(word => word[index] == Constants.BLOCKED);
        public ChoiceStep MakeChoice()
        {
            if (MaxPossibilities <= 1) throw new InvalidOperationException("Cannot call MakeChoice on entity with 1 or less options");

            if (Preferred.Length == 1 && Obscure.Length == 1)
            {
                return new ChoiceStep
                {
                    Choice = new Definite(new(Preferred[0], [Preferred[0]])),
                    Remaining = new Definite(new(Obscure[0], [Obscure[0]])),
                };
            }

            var (pref1, pref2) = Preferred.Split();
            var (obsc1, obsc2) = Obscure.Split();

            IPossibleLines choice = (pref1, obsc1) switch
            {
                ({ Length: 0 }, { Length: 0 }) => Impossible.Instance(NumLetters),
                ({ Length: 1 }, { Length: 0 }) => new Definite(new ConcreteLine(pref1[0], [pref1[0]])),
                ({ Length: 0 }, { Length: 1 }) => new Definite(new ConcreteLine(obsc1[0], [obsc1[0]])),
                _ => new Words(pref1, obsc1)
            };
            IPossibleLines remaining = (pref2, obsc2) switch
            {
                ({ Length: 0 }, { Length: 0 }) => Impossible.Instance(NumLetters),
                ({ Length: 1 }, { Length: 0 }) => new Definite(new ConcreteLine(pref2[0], [pref2[0]])),
                ({ Length: 0 }, { Length: 1 }) => new Definite(new ConcreteLine(obsc2[0], [obsc2[0]])),
                _ => new Words(pref2, obsc2)
            };

            return new ChoiceStep
            {
                Choice = choice,
                Remaining = remaining,
            };
        }

        public IPossibleLines Filter(CharSet constraint, int index)
        {
            if (constraint.IsFull) return this;

            var filteredPreferred = Preferred.RemoveAll(word => !constraint.Contains(word[index]));
            var filteredObscure = Obscure.RemoveAll(word => constraint.Contains(word[index]));

            return (filteredPreferred, filteredObscure) switch
            {
                ({ Length: 0 }, { Length: 0 }) => Impossible.Instance(NumLetters),
                _ => new Words(filteredPreferred, filteredObscure),
            };
        }
        public IPossibleLines Filter(char constraint, int index)
        {
            var filteredPreferred = Preferred.RemoveAll(word => constraint != word[index]);
            var filteredObscure = Obscure.RemoveAll(word => constraint != word[index]);

            return (filteredPreferred, filteredObscure) switch
            {
                ({ Length: 0 }, { Length: 0 }) => Impossible.Instance(NumLetters),
                _ => new Words(filteredPreferred, filteredObscure),
            };
        }
        public IPossibleLines RemoveWordOption(string word)
        {
            if (word.Length != NumLetters) return this;

            var p = Preferred.Remove(word);
            var o = Obscure.Remove(word);

            if (p.Length == Preferred.Length && o.Length == Obscure.Length) return this;
            if (p.Length == 0 && o.Length == 0) return Impossible.Instance(NumLetters);
            return this with { Preferred = p, Obscure = o };
        }
        public IEnumerable<ConcreteLine> Iterate() =>
            Preferred.Concat(Obscure)
            .Select(w => new ConcreteLine(w, [w]));
        public ConcreteLine? FirstOrNull()
        {
            if (Preferred.Count > 0) return new ConcreteLine(Preferred[0], [Preferred[0]]);
            if (Obscure.Count > 0) return new ConcreteLine(Obscure[0], [Obscure[0]]);
            return null;
        }
    }
    record BlockBefore(IPossibleLines Lines) : IPossibleLines
    {
        public int NumLetters => 1 + Lines.NumLetters;
        public long MaxPossibilities => Lines.MaxPossibilities;
        public void CharsAt(CharSet accumulate, int index)
        {
            if (accumulate.IsFull) return;
            if (index == 0) accumulate.Add(Constants.BLOCKED);
            else Lines.CharsAt(accumulate, index - 1);
        }
        public bool DefinitelyBlockedAt(int index)
        {
            if (index == 0) return true;
            else return Lines.DefinitelyBlockedAt(index - 1);
        }
        public IPossibleLines Filter(CharSet constraint, int index)
            => index switch
            {
                0 => constraint.Contains(Constants.BLOCKED) ? this : Impossible.Instance(NumLetters),
                _ => Lines.Filter(constraint, index - 1) switch
                {
                    Impossible => Impossible.Instance(NumLetters),
                    IPossibleLines l => this with { Lines = l },
                },
            };
        public IPossibleLines Filter(char constraint, int index)
            => index switch
            {
                0 => constraint == Constants.BLOCKED ? this : Impossible.Instance(NumLetters),
                _ => Lines.Filter(constraint, index - 1) switch
                {
                    Impossible => Impossible.Instance(NumLetters),
                    IPossibleLines l => this with { Lines = l },
                },
            };
        public IPossibleLines RemoveWordOption(string word)
        {
            if (word.Length >= Lines.NumLetters) return this;
            var lines = Lines.RemoveWordOption(word);

            if (lines == Lines) return this;
            if (lines is Impossible) return Impossible.Instance(NumLetters);
            return this with { Lines = lines };
        }
        public IEnumerable<ConcreteLine> Iterate() => Lines.Iterate().Select(line => line with { Line = Constants.BLOCKED + line.Line });
        public ConcreteLine? FirstOrNull() => Lines.FirstOrNull() switch
        {
            null => null,
            ConcreteLine line => line with { Line = Constants.BLOCKED + line.Line },
        };
        public ChoiceStep MakeChoice()
        {
            ChoiceStep inner = Lines.MakeChoice();
            return new ChoiceStep
            {
                Choice = this with { Lines = inner.Choice },
                Remaining = this with { Lines = inner.Remaining }
            };
        }
    }
    record BlockAfter(IPossibleLines Lines) : IPossibleLines
    {
        public int NumLetters => 1 + Lines.NumLetters;
        public long MaxPossibilities => Lines.MaxPossibilities;
        public void CharsAt(CharSet accumulate, int index)
        {
            if (accumulate.IsFull) return;
            if (index == NumLetters - 1) accumulate.Add(Constants.BLOCKED);
            else Lines.CharsAt(accumulate, index);
        }
        public bool DefinitelyBlockedAt(int index)
        {
            if (index == NumLetters - 1) return true;
            else return Lines.DefinitelyBlockedAt(index);
        }
        public IPossibleLines Filter(CharSet constraint, int index)
            => index switch
            {
                _ when index == NumLetters - 1 => constraint.Contains(Constants.BLOCKED) ? this : Impossible.Instance(NumLetters),
                _ => Lines.Filter(constraint, index) switch
                {
                    Impossible => Impossible.Instance(NumLetters),
                    IPossibleLines l => this with { Lines = l },
                },
            };
        public IPossibleLines Filter(char constraint, int index)
            => index switch
            {
                _ when index == NumLetters - 1 => constraint == Constants.BLOCKED ? this : Impossible.Instance(NumLetters),
                _ => Lines.Filter(constraint, index) switch
                {
                    Impossible => Impossible.Instance(NumLetters),
                    IPossibleLines l => this with { Lines = l },
                },
            };
        public IPossibleLines RemoveWordOption(string word)
        {
            if (word.Length >= Lines.NumLetters) return this;
            var lines = Lines.RemoveWordOption(word);

            if (lines == Lines) return this;
            if (lines is Impossible) return Impossible.Instance(NumLetters);
            return this with { Lines = lines };
        }
        public IEnumerable<ConcreteLine> Iterate() => Lines.Iterate().Select(line => line with { Line = line.Line + Constants.BLOCKED });
        public ConcreteLine? FirstOrNull() => Lines.FirstOrNull() switch
        {
            null => null,
            ConcreteLine line => line with { Line = line.Line + Constants.BLOCKED },
        };
        public ChoiceStep MakeChoice()
        {
            ChoiceStep inner = Lines.MakeChoice();
            return new ChoiceStep
            {
                Choice = this with { Lines = inner.Choice },
                Remaining = this with { Lines = inner.Remaining }
            };
        }
    }
    record BlockBetween(IPossibleLines First, IPossibleLines Second) : IPossibleLines
    {
        public int NumLetters => 1 + First.NumLetters + Second.NumLetters;
        public long MaxPossibilities => First.MaxPossibilities * Second.MaxPossibilities;
        public void CharsAt(CharSet accumulate, int index)
        {
            if (accumulate.IsFull) return;
            if (index == First.NumLetters) accumulate.Add(Constants.BLOCKED);
            else if (index < First.NumLetters) First.CharsAt(accumulate, index);
            else Second.CharsAt(accumulate, index - (First.NumLetters + 1));
        }
        public bool DefinitelyBlockedAt(int index)
        {
            if (index == First.NumLetters) return true;
            else if (index < First.NumLetters) return First.DefinitelyBlockedAt(index);
            else return Second.DefinitelyBlockedAt(index - (First.NumLetters + 1));
        }
        public IPossibleLines Filter(CharSet constraint, int index)
        {
            if (constraint.IsFull) return this;
            if (index == First.NumLetters) return constraint.Contains(Constants.BLOCKED) ? this : Impossible.Instance(NumLetters);

            var first = index < First.NumLetters ? First.Filter(constraint, index) : First;
            var second = index > First.NumLetters ? Second.Filter(constraint, index - (1 + First.NumLetters)) : Second;

            if (first is Impossible || second is Impossible) return Impossible.Instance(NumLetters);
            var combo = new BlockBetween(first, second);

            if (combo.MaxPossibilities < 50)
            {
                var choices = combo.Iterate()
                    .DistinctBy(l => l.Line)
                    .Select(l => new Definite(l))
                    .ToImmutableArray<IPossibleLines>();
                return choices switch
                {
                    { IsEmpty: true } => new Impossible(NumLetters),
                    _ => new Compound(choices)
                };
            }
            else return combo;
        }
        public IPossibleLines Filter(char constraint, int index)
        {
            if (index == First.NumLetters) return constraint == Constants.BLOCKED ? this : Impossible.Instance(NumLetters);

            var first = index < First.NumLetters ? First.Filter(constraint, index) : First;
            var second = index > First.NumLetters ? Second.Filter(constraint, index - (1 + First.NumLetters)) : Second;

            if (first is Impossible || second is Impossible) return Impossible.Instance(NumLetters);
            return new BlockBetween(first, second);
        }
        public IPossibleLines RemoveWordOption(string word)
        {
            if (word.Length > First.NumLetters && word.Length > Second.NumLetters) return this;

            var first = word.Length > First.NumLetters ?
                First : First.RemoveWordOption(word);
            var second = word.Length > Second.NumLetters ?
                Second : Second.RemoveWordOption(word);

            if (first == First && second == Second) return this;
            if (first is Impossible || second is Impossible) return Impossible.Instance(NumLetters);
            return this with { First = first, Second = second };
        }
        public IEnumerable<ConcreteLine> Iterate() =>
            First.Iterate()
            .SelectMany(first => Second.Iterate()
                .Where(second => !second.Words.Intersect(first.Words).Any())
                .Select(second => new ConcreteLine(
                    Line: $"{first.Line}{Constants.BLOCKED}{second.Line}",
                    Words: first.Words.AddRange(second.Words))
                )
            );
        public ConcreteLine? FirstOrNull() => Iterate().Take(1).Select(l => (ConcreteLine?)l).FirstOrDefault();
        public ChoiceStep MakeChoice()
        {
            if (First.MaxPossibilities > 1)
            {
                ChoiceStep firstChoice = First.MakeChoice();
                return new ChoiceStep
                {
                    Choice = this with { First = firstChoice.Choice },
                    Remaining = this with { First = firstChoice.Remaining }
                };
            }

            ChoiceStep secondChoice = Second.MakeChoice();
            return new ChoiceStep
            {
                Choice = this with { Second = secondChoice.Choice },
                Remaining = this with { Second = secondChoice.Remaining }
            };
        }
    }
    record Compound(ImmutableArraySegment<IPossibleLines> Possibilities) : IPossibleLines
    {
        public int NumLetters => Possibilities[0].NumLetters;
        public long MaxPossibilities => Possibilities.Sum(x => x.MaxPossibilities);
        public void CharsAt(CharSet accumulate, int index)
        {
            foreach (var line in Possibilities)
            {
                line.CharsAt(accumulate, index);
                if (accumulate.IsFull) return;
            }
        }
        public bool DefinitelyBlockedAt(int index)
            => Possibilities.All(p => p.DefinitelyBlockedAt(index));

        public IPossibleLines Filter(CharSet constraint, int index)
        {
            if (constraint.IsFull) return this;

            var filtered = Possibilities
                .Select(possible => possible.Filter(constraint, index))
                .Where(possible => possible is not Impossible && possible.MaxPossibilities > 0)
                .ToImmutableArray();

            if (filtered.Length == 0) return Impossible.Instance(NumLetters);
            else if (filtered.Length == 1) return filtered[0];
            return new Compound(filtered);
        }
        public IPossibleLines Filter(char constraint, int index)
        {
            var filtered = Possibilities
                .Select(possible => possible.Filter(constraint, index))
                .Where(possible => possible is not Impossible && possible.MaxPossibilities > 0)
                .ToImmutableArray();

            if (filtered.Length == 0) return Impossible.Instance(NumLetters);
            else if (filtered.Length == 1) return filtered[0];

            if (filtered.Sum(p => p.MaxPossibilities) <= 20 && filtered.Any(p => p is not Definite))
            {
                filtered = [.. filtered
                    .SelectMany(p => p.Iterate())
                    .Distinct()
                    .Select(line => new Definite(line))];
            }

            if (filtered.Length == 0) return Impossible.Instance(NumLetters);
            else if (filtered.Length == 1) return filtered[0];

            return new Compound(filtered);
        }
        public IPossibleLines RemoveWordOption(string word)
        {
            if (word.Length > NumLetters) return this;

            var filtered = Possibilities
                .Select(possible => possible.RemoveWordOption(word))
                .Where(possible => possible is not Impossible)
                .ToImmutableArray();

            if (filtered.Length == 0) return Impossible.Instance(NumLetters);
            else if (filtered.Length == 1) return filtered[0];

            return new Compound(filtered);
        }
        public IEnumerable<ConcreteLine> Iterate() => Possibilities.SelectMany(possible => possible.Iterate());
        public ConcreteLine? FirstOrNull() => Possibilities.Select(possible => possible.FirstOrNull()).Where(p => p != null).FirstOrDefault();
        public ChoiceStep MakeChoice()
        {
            if (Possibilities.Length <= 1) throw new InvalidOperationException("Cannot make a choice if Possibilities <= 1");
            if (MaxPossibilities <= 1) throw new InvalidOperationException("Cannot make a choice if MaxPossibilities <= 1");

            var (choice, remaining) = Possibilities.Split();

            return new ChoiceStep
            {
                Choice = choice.Length switch
                {
                    0 => new Impossible(NumLetters),
                    1 => choice[0],
                    _ => this with { Possibilities = choice },
                },
                Remaining = remaining.Length switch
                {
                    0 => new Impossible(NumLetters),
                    1 => remaining[0],
                    _ => this with { Possibilities = remaining },
                }
            };
        }
    }
    record Definite(ConcreteLine Line) : IPossibleLines
    {
        public int NumLetters => Line.Line.Length;
        public long MaxPossibilities => 1;
        public void CharsAt(CharSet accumulate, int index)
        {
            accumulate.Add(Line.Line[index]);
        }
        public bool DefinitelyBlockedAt(int index) => Line.Line[index] == Constants.BLOCKED;
        public IPossibleLines Filter(CharSet constraint, int index)
            => constraint.Contains(Line.Line[index]) ? this : Impossible.Instance(NumLetters);
        public IPossibleLines Filter(char constraint, int index)
            => constraint == Line.Line[index] ? this : Impossible.Instance(NumLetters);
        public IPossibleLines RemoveWordOption(string word)
        {
            if (word.Length > Line.Line.Length) return this;

            if (Line.Words.Contains(word)) return Impossible.Instance(NumLetters);
            return this;
        }
        public IEnumerable<ConcreteLine> Iterate()
        {
            yield return Line;
        }
        public ConcreteLine? FirstOrNull() => Line;
        public ChoiceStep MakeChoice() => throw new InvalidOperationException("Cannot make a choice on a definite line");
    }

}

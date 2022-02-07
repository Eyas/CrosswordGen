using Crossword;
using System.Collections.Immutable;

namespace WordGenLib
{
    public record class FinalGrid(ImmutableArray<string> Down, ImmutableArray<string> Across)
    {
        public string Repr => string.Join('\n', Down);
    }

    public class CharSet
    {
        private readonly bool[] _available;
        private readonly char _min;
        private int _ct;

        public CharSet(char min, char max)
        {
            _min = min;
            _available = new bool[1 + (max - min)];
            _ct = 0;
        }

        public CharSet() : this('`', 'z') { }

        public void Add(char c)
        {
            if (!_available[c - _min])
            {
                _ct += 1;
                _available[ c - _min ] = true;
            }
        }

        public bool Contains(char c)
        {
            return _available[c - _min ];
        }

        public bool IsFull => _ct == _available.Length;
    }
    public class GridDictionary<T> where T : notnull, new()
    {
        private readonly T?[,] _values;
        public GridDictionary(int size)
        {
            _values = new T?[size, size];
        }

        public T GetOrAddDefault((int x, int y) kv)
        {
            var x = _values[kv.x, kv.y];
            if (x == null)
            {
                x = new();
                _values[kv.x, kv.y] = x;
            }
            return x;
        }

        public T this[int x, int y]
        {
            get
            {
                var v = _values[x, y];
                System.Diagnostics.Debug.Assert(v != null);
                return v;
            }
        }

    }

    public class Generator
    {
        private readonly int gridSize;

        private readonly Dictionary<int, ImmutableArray<string>> commonWordsByLength;
        private readonly Dictionary<int, ImmutableArray<string>> obscureWordsByLength;

        private readonly IPossibleLines possibleLines;

        public int GridSize => gridSize;

        public static Generator Create(
            int gridSize,
            ImmutableArray<string>? commonWords = null,
            ImmutableArray<string>? obscureWords = null
        )
        {
            return new(
                gridSize,
                commonWords ?? GridReader.COMMON_WORDS,
                obscureWords ?? GridReader.OBSCURE_WORDS
                );
        }

        internal Generator(int gridSize, ImmutableArray<string> commonWords, ImmutableArray<string> obscureWords)
        {
            this.gridSize = gridSize;

            commonWordsByLength = commonWords
                .RemoveAll(s => s.Length <= 2 || s.Length > gridSize)
                .GroupBy(w => w.Length)
                .ToDictionary(g => g.Key, g => g.ToImmutableArray ());
            obscureWordsByLength = obscureWords
                .RemoveAll(s => s.Length <= 2 || s.Length > gridSize)
                .Except(commonWords)
                .GroupBy(w => w.Length)
                .ToDictionary(g => g.Key, g => g.ToImmutableArray());

            for (int i = 3; i <= gridSize; ++i)
            {
                if (!commonWordsByLength.ContainsKey(i)) commonWordsByLength[i] = ImmutableArray<string>.Empty;
                if (!obscureWordsByLength.ContainsKey(i)) obscureWordsByLength[i] = ImmutableArray<string>.Empty;
            }

            possibleLines = AllPossibleLines(gridSize);
        }

        interface IPossibleLines
        {
            int NumLetters { get; }
            long MaxPossibilities { get; }
            void CharsAt(CharSet accumulate, int index);
            bool DefinitelyBlockedAt(int index);
            IPossibleLines Filter(CharSet constraint, int index);
            IPossibleLines Filter(char constraint, int index);
            IEnumerable<string> Iterate();
        }
        record Impossible(int NumLetters) : IPossibleLines
        {
            public long MaxPossibilities => 0;
            public void CharsAt(CharSet accumulate, int _) { }
            public bool DefinitelyBlockedAt(int _) => false;
            public IPossibleLines Filter(CharSet _, int _2) => this;
            public IPossibleLines Filter(char _, int _2) => this;
            public IEnumerable<string> Iterate() => Enumerable.Empty<string>();

            private static readonly Impossible?[] _cache = new Impossible?[25];
            public static Impossible Instance(int numLetters)
            {
                if (_cache[numLetters] == null) _cache[numLetters] = new Impossible(numLetters);

                return _cache[numLetters]!;
            }
        }
        record Words(ImmutableArray<string> Preferred, ImmutableArray<string> Obscure) : IPossibleLines
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

            public IPossibleLines Filter(CharSet constraint, int index)
            {
                if (constraint.IsFull) return this;

                var filteredPreferred = Preferred.Where(word => constraint.Contains(word[index])).ToImmutableArray();
                var filteredObscure = Obscure.Where(word => constraint.Contains(word[index])).ToImmutableArray();

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
            public IEnumerable<string> Iterate() => Preferred.Concat(Obscure);
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
            public IEnumerable<string> Iterate() => Lines.Iterate().Select(word => Constants.BLOCKED + word);
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
            public IEnumerable<string> Iterate() => Lines.Iterate().Select(word => word + Constants.BLOCKED);
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
                    return new Words(Preferred: combo.Iterate().Distinct().ToImmutableArray(), ImmutableArray<string>.Empty);
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
            public IEnumerable<string> Iterate() => First.Iterate().SelectMany(first => Second.Iterate().Select(second => $"{first}{Constants.BLOCKED}{second}"));
        }
        record Compound(ImmutableArray<IPossibleLines> Possibilities) : IPossibleLines
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
                    .Where(possible => possible is not Impossible)
                    .ToImmutableArray();

                if (filtered.Length == 0) return Impossible.Instance(NumLetters);
                else if (filtered.Length == 1) return filtered[0];
                return new Compound(filtered);
            }
            public IPossibleLines Filter(char constraint, int index)
            {
                var filtered = Possibilities
                    .Select(possible => possible.Filter(constraint, index))
                    .Where(possible => possible is not Impossible)
                    .ToImmutableArray();

                if (filtered.Length == 0) return Impossible.Instance(NumLetters);
                else if (filtered.Length == 1) return filtered[0];

                if (filtered.Sum(p => p.MaxPossibilities) <= 20 && filtered.Any(p => p is not Definite))
                {
                    filtered = filtered
                        .SelectMany(p => p.Iterate())
                        .Distinct()
                        .Select(line => new Definite(line))
                        .ToImmutableArray<IPossibleLines>();
                }

                return new Compound(filtered);
            }
            public IEnumerable<string> Iterate() => Possibilities.SelectMany(possible => possible.Iterate());
        }
        record Definite(string Line) : IPossibleLines
        {
            public int NumLetters => Line.Length;
            public long MaxPossibilities => 1;
            public void CharsAt(CharSet accumulate, int index)
            {
                accumulate.Add(Line[index]);
            }
            public bool DefinitelyBlockedAt(int index) => Line[index] == Constants.BLOCKED;
            public IPossibleLines Filter(CharSet constraint, int index)
                => constraint.Contains(Line[index]) ? this : Impossible.Instance(NumLetters);
            public IPossibleLines Filter(char constraint, int index)
                => constraint == Line[index] ? this : Impossible.Instance(NumLetters);
            public IEnumerable<string> Iterate()
            {
                yield return Line;
            }
        }

        private record GridState(ImmutableArray<IPossibleLines> Down, ImmutableArray<IPossibleLines> Across) {
            public int SideLength => Down.Length;
            public int Area => SideLength * SideLength;

            public int? UndecidedDown
                => Down.Select((options, index) => (options, index)).Where(oi => oi.options.MaxPossibilities > 1).OrderBy(oi => oi.options.MaxPossibilities).Select(oi => (int?)oi.index).FirstOrDefault();
            public int? UndecidedAcross
                => Across.Select((options, index) => (options, index)).Where(oi => oi.options.MaxPossibilities > 1).OrderBy(oi => oi.options.MaxPossibilities).Select(oi => (int?)oi.index).FirstOrDefault();
        }
        interface IPruneStrategy
        {
            record class PruneStepState(int MaxLength) { }
            bool ShouldKeepCommonWord(PruneStepState state);
            bool ShouldKeepObscureWord(PruneStepState state);
        }
        class DontPrune : IPruneStrategy
        {
            public bool ShouldKeepCommonWord(IPruneStrategy.PruneStepState _) => true;
            public bool ShouldKeepObscureWord(IPruneStrategy.PruneStepState _) => true;
        }
        class PruneRoots : IPruneStrategy
        {
            public bool ShouldKeepCommonWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > 1;
            public bool ShouldKeepObscureWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > Math.Min(2, s.MaxLength - 2);
        }
        class PruneAggressive : IPruneStrategy
        {
            public bool ShouldKeepCommonWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > 1;
            public bool ShouldKeepObscureWord(IPruneStrategy.PruneStepState s) => Random.Shared.Next(s.MaxLength) > Math.Min(3, s.MaxLength - 2);
        }


        private IPossibleLines AllPossibleLines(int maxLength)
        {
            return AllPossibleLines(maxLength, new Dictionary<int, IPossibleLines>(), new DontPrune());
        }
        private IPossibleLines AllPossibleLines(int maxLength, Dictionary<int, IPossibleLines> memo, IPruneStrategy prune)
        {
            if (maxLength > gridSize) throw new Exception($"{nameof(maxLength)} ({maxLength}) cannot be greater than {nameof(gridSize)} {gridSize}");
            if (maxLength < 3) return Impossible.Instance(maxLength);

            if (memo.TryGetValue(maxLength, out var result))
            {
                return result;
            }
            var pruneState = new IPruneStrategy.PruneStepState(maxLength);

            var compoundBuilder = ImmutableArray.CreateBuilder<IPossibleLines>();

            {
                var possibleWords = new Words(
                    Preferred:
                        commonWordsByLength[maxLength]
                            .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                            .Where(_ => prune.ShouldKeepCommonWord(pruneState))
                            .ToImmutableArray(),
                    Obscure:
                        obscureWordsByLength[maxLength]
                            .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                            .Where(_ => prune.ShouldKeepObscureWord(pruneState))
                            .ToImmutableArray()
                    );
                compoundBuilder.Add(possibleWords);
            }

            // recurse into all combination of [ANYTHING]*[ANYTHING]
            //
            // For length 10:
            // 0 1 2 3 4 5 6 7 8 9
            // _ _ _ _ _ _ _ _ _ _
            //       ^     ^
            // Blockage can be anywhere etween idx 3 and len-4 (inclusive).
            if (maxLength >= 7)
            {
                compoundBuilder.AddRange(
                    Enumerable.Range(start: 3, count: maxLength - 6)
                        .Select(i =>
                        {
                            int firstLength = i;  // Always >= 3.
                        int secondLength = maxLength - (i + 1);  // Always >= 3.

                        return (firstLength, secondLength);
                        })
                        .OrderBy(_ => Random.Shared.Next(int.MaxValue))
                        .Select(l => new BlockBetween(
                            AllPossibleLines(l.firstLength, memo, prune),
                            AllPossibleLines(l.secondLength, memo, prune)
                            ))
                    );
            }

            // recurse into *[ANYTHING], and [ANYTHING]*
            {
                var smaller = AllPossibleLines(maxLength - 1, memo, prune);
                if (smaller is not Impossible)
                {
                    compoundBuilder.AddRange(
                        new IPossibleLines[] {
                        new BlockBefore(smaller),
                            new BlockAfter(smaller)
                        }.OrderBy(_ => Random.Shared.Next(int.MaxValue))
                    );
                }
            }

            var options = compoundBuilder switch
            {
                { Count: 0 } => Impossible.Instance(maxLength),
                { Count: 1 } => compoundBuilder[0],
                _ => new Compound(compoundBuilder.ToImmutable()),
            };

            memo.Add(maxLength, options);
            return options;
        }

        static GridState Prefilter(GridState state, Direction direction)
        {
            var toFilter = direction == Direction.Horizontal ? state.Across : state.Down;
            var constraint = direction == Direction.Horizontal ? state.Down : state.Across;

            if (toFilter.Any(r => r.MaxPossibilities == 0) || constraint.Any(c => c.MaxPossibilities == 0)) return state;

            // x and y here are abstracted wlog based on toFilter/constraint, not truly
            // connected to Horizontal vs Vertical.
            GridDictionary<CharSet> available = new(state.Across.Length);
            for (int x = 0; x < constraint.Length; x++)
            {
                var c = constraint[x];

                for (int y = 0; y < c.NumLetters; ++y)
                {
                    var chars = available.GetOrAddDefault((x, y));
                    c.CharsAt(chars, y);
                }
            }

            var filtered = toFilter.Select((possibles, y) =>
            {
                if (Enumerable.Range(0, toFilter.Length)
                    .All(x => available[x, y].IsFull)) return possibles;

                for (int x = 0; x < possibles.NumLetters; ++x)
                {
                    possibles = possibles.Filter(available[x, y], x);
                }
                return possibles;
            }).ToImmutableArray();

            if (direction == Direction.Horizontal)
            {
                return state with { Across = filtered };
            }
            else return state with { Down = filtered };
        }

        private static IEnumerable<FinalGrid> AllPossibleGrids(GridState root)
        {
            // If we are at a point in our tree some row/column is unfillable, prune this tree.
            if (root.Down.Any(options => options.MaxPossibilities == 0)) yield break;
            if (root.Across.Any(options => options.MaxPossibilities == 0)) yield break;

            // If board is > 35% blocked, it's not worth iterating in it.
            int numDefinitelyBlocked = root.Down
                .Where(options => options.MaxPossibilities == 1)
                .Sum(options => options.Iterate().First().Count(c => c == Constants.BLOCKED) );
            if (numDefinitelyBlocked > (root.Area * 0.35) )
            {
                yield break;
            }

            // Prefilter
            {
                int tries = 0;
                Direction direction = Direction.Horizontal;
                while (tries < 4)
                {
                    ++tries;
                    GridState newState = Prefilter(root, direction);
                    if (!Changed(root, newState) && tries > 1) break;

                    root = newState;
                    direction = direction == Direction.Vertical ? Direction.Horizontal : Direction.Vertical;
                }

                // If we are at a point in our tree some row/column is unfillable, prune this tree.
                if (root.Down.Any(options => options.MaxPossibilities == 0)) yield break;
                if (root.Across.Any(options => options.MaxPossibilities == 0)) yield break;
            }

            // If board is entirely divided, s.t. no word spans two "halves" of the
            // board, we want to stop.
            //
            // We already can't have entire blocked lines. But we can have:
            // _ _ _ ` ` `
            // ` ` ` _ _ _
            //
            // This can still be better, e.g. it doesn't account for a "quadrant"
            // being cordoned off.
            bool[] previouslyBlocked = new bool[root.Down.Length];

            foreach (var col in root.Down)
            {
                bool anyBlocked = false;
                for (int i = 0; i < root.Across.Length; i++)
                {
                    if (!col.DefinitelyBlockedAt(i)) continue;
                    anyBlocked = true;
                    previouslyBlocked[i] = true;
                }

                if (!anyBlocked)
                {
                    Array.Fill(previouslyBlocked, false);
                    continue;
                }

                if (previouslyBlocked.All(v => v == true)) yield break;
            }

            int? undecidedDown = root.UndecidedDown;
            int? undecidedAcross = root.UndecidedAcross;

            if (undecidedDown == null && undecidedAcross == null)
            {
                var down = root.Down.Select(col => col.Iterate().First()).ToImmutableArray();
                var across = root.Across.Select(row => row.Iterate().First()).ToImmutableArray();

                if (down.Zip(across).Any(both => both.First == both.Second))
                    yield break;

                yield return new FinalGrid(
                    Down: down,
                    Across: across
                );
                yield break;
            }

            var possibleGrids = (undecidedDown, undecidedAcross) switch
            {
                (int uD, null) => AllPossibleGrids(root, uD, Direction.Vertical),
                (int uD, int uA) when root.Down[uD].MaxPossibilities < root.Across[uA].MaxPossibilities => AllPossibleGrids(root, uD, Direction.Vertical),
                (null, int uA) => AllPossibleGrids(root, uA, Direction.Horizontal),
                (int uD, int uA) when root.Down[uD].MaxPossibilities >= root.Across[uA].MaxPossibilities => AllPossibleGrids(root, uA, Direction.Horizontal),
                _ => Enumerable.Empty<FinalGrid>(),
            };
            foreach (var final in possibleGrids) yield return final;
        }

        private static IEnumerable<FinalGrid> AllPossibleGrids(GridState root, int index, Direction dir)
        {
            var optionAxis = (dir == Direction.Horizontal ? root.Across : root.Down);
            var oppositeAxis = (dir == Direction.Horizontal ? root.Down : root.Across);

            // Trim situations where horizontal and vertal words are same.
            for (int i = 0; i < optionAxis.Length; ++i)
            {
                if (optionAxis[i].MaxPossibilities > 1) continue;
                if (oppositeAxis[i].MaxPossibilities > 1) continue;

                if (optionAxis[i].Iterate().First() == oppositeAxis[i].Iterate().First()) yield break;
            }

            var options = optionAxis[index];

            // The below loop "makes decisions" and recurses. If we already
            // have one attempt, that means it's already pre-decided.
            if (options.MaxPossibilities == 1) yield break;

            foreach (var attempt in options.Iterate())
            {
                var attemptOpposite = oppositeAxis.ToArray();

                for (int i = 0; i < attempt.Length; i++)
                {
                    // WLOG say we dir is Horizontal, and opopsite is Vertical.
                    // we have:
                    //
                    // W O R D
                    // _ _ _ _
                    // _ _ _ _
                    // _ _ _ _
                    //
                    // Then go over each COL (i), filtering s.t. possible lines
                    // only include cases where col[i]'s |attempt|th character == attempt[i].
                    var constriant = attempt[i];

                    attemptOpposite[i] = attemptOpposite[i].Filter(constriant, index);
                    if (attemptOpposite[i].MaxPossibilities == 1 && attemptOpposite[i].Iterate().First() == attempt) yield break;
                }

                if (attemptOpposite.All(opts => opts is not Impossible))
                {
                    var oppositeFinal = attemptOpposite.ToImmutableArray();
                    var optionFinal = optionAxis.Select((regular, idx) => idx == index ? new Definite(attempt) : regular).ToImmutableArray();

                    if (attemptOpposite.Zip(optionFinal)
                        .Where(ab => ab.First.MaxPossibilities == 1 && ab.Second.MaxPossibilities == 1)
                        .Any(ab => ab.First.Iterate().First() == ab.Second.Iterate().First()))
                        yield break;

                    var newRoot = (dir == Direction.Horizontal) ?
                        new GridState(
                            Down: oppositeFinal,
                            Across: optionFinal
                            ) :
                        new GridState(
                            Down: optionFinal,
                            Across: oppositeFinal
                            );

                    foreach (var final in AllPossibleGrids(newRoot)) yield return final;
                }
            }
        }

        static bool Changed(GridState before, GridState after)
        {
            for (int i = 0; i < before.Down.Length; ++i)
            {
                if (before.Down[i].MaxPossibilities != after.Down[i].MaxPossibilities) return true;
                if (before.Across[i].MaxPossibilities != after.Across[i].MaxPossibilities) return true;
            }
            return false;
        }

        public IEnumerable<FinalGrid> PossibleGrids()
        {
            GridState state = new(
                Down: Enumerable.Range(0, gridSize).Select(_ => possibleLines).ToImmutableArray(),
                Across: Enumerable.Range(0, gridSize).Select(_ => possibleLines).ToImmutableArray()
            );
            return AllPossibleGrids(state).DistinctBy(x => x.Repr);
        }

        public IEnumerable<FinalGrid> PossibleGridWithConstraints(char[,] constraints)
        {
            if (constraints == null) throw new ArgumentNullException(nameof(constraints));
            if (constraints.GetLength(0) != gridSize || constraints.GetLength(1) != gridSize) throw new ArgumentException($"{nameof(constraints)} should have size {GridSize} x {GridSize}");

            string[] acrossTemplates = Enumerable.Range(0, gridSize).Select(y => string.Join("", Enumerable.Range(0, gridSize).Select(x => constraints[x, y]))).ToArray();
            string[] downTemplates = Enumerable.Range(0, gridSize).Select(x => string.Join("", Enumerable.Range(0, gridSize).Select(y => constraints[x, y]))).ToArray();

            GridState state = new(
                Down: Enumerable.Range(0, gridSize).Select(i => CompatibleLines(downTemplates[i])).ToImmutableArray(),
                Across: Enumerable.Range(0, gridSize).Select(i => CompatibleLines(acrossTemplates[i])).ToImmutableArray()
            );
            return AllPossibleGrids(state).DistinctBy(x => x.Repr);
        }

        private IPossibleLines CompatibleLines(string template)
        {
            if (template.All(x => x == ' ')) return possibleLines;
            if (template.All(x => x != ' ')) return new Definite(template);

            var lines = possibleLines;
            for (int i = 0; i < template.Length; i++)
            {
                if (template[i] == ' ') continue;
                lines = lines.Filter(template[i], i);
            }
            return lines;
        }

        public  IEnumerable<string> CompatibleLineStrings(string template)
        {
            return CompatibleLines(template).Iterate();
        }
    }

    internal static class GenHelper
    {
        public static void FillRow(char?[,] grid, int row, string word)
        {
            for (int i = 0; i < word.Length; ++i)
            {
                if (word[i] == ' ') continue;
                grid[i, row] = word[i];
            }
        }
    }

    public static class GridReader
    {
        internal static ImmutableArray<string> COMMON_WORDS =
            Properties.Resources.words.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None)
            .Concat(Properties.Resources.phrases.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None))
            .Select(s => s.Trim().Replace(" ", ""))
            .Distinct()
            .ToImmutableArray();

        internal static ImmutableArray<string> OBSCURE_WORDS =
            Properties.Resources.phrases.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None)
            .Concat(Properties.Resources.wikipedia.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None))
            .Concat(Properties.Resources.from_lexems.Split(new[] { "\r\n", "\r", "\n" }, StringSplitOptions.None))
            .Select(s => s.Trim().Replace(" ", ""))
            .Distinct()
            .Except(COMMON_WORDS)
            .ToImmutableArray();

        internal static ImmutableArray<string> ALL_WORDS = COMMON_WORDS.AddRange(OBSCURE_WORDS);

        public static HashSet<string> AllowedWords()
        {
            return new HashSet<string>(ALL_WORDS);
        }

        public static string[] DownWords(FinalGrid grid)
        {
            return grid.Down.SelectMany(s => s.Split(Constants.BLOCKED)).Where(s => s.Length > 0).ToArray();
        }

        public static string[] AcrossWords(FinalGrid grid)
        {
            return grid.Across.SelectMany(s => s.Split(Constants.BLOCKED)).Where(s => s.Length > 0).ToArray();
        }
    }

    public enum Direction { Horizontal, Vertical }
}
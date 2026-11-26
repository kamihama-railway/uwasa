-- Formal model of the Uwasa VM execution state machine.
-- Mirrors vm.go / bytecode.go / vm_compiler.go (branch go-perf-opt).
-- Goal: prove semantics preservation of peephole fusions and analyse,
-- with an explicit cost model, whether they can improve performance.

import Std

namespace UWasa

-- ---------------------------------------------------------------------------
-- 1. Values and the (abstract) operation semantics
-- ---------------------------------------------------------------------------

inductive Val where
  | nil
  | int (i : Int)
  | bool (b : Bool)
  | str (s : String)
  deriving Inhabited, BEq, DecidableEq, Repr

-- Operations are abstract: every fusion below uses exactly the same operation
-- functions as the unfused instruction sequence, so the fusion proofs hold for
-- ANY implementation of them (including the float/coercion paths in Go).
structure Ops where
  add : Val → Val → Val
  eqv : Val → Val → Bool
  gt : Val → Val → Bool
  lt : Val → Val → Bool
  truthy : Val → Bool
  -- Go's isValTruthy: truthy(ValBool b) = b  (vm.go:643-647)
  truthy_bool : ∀ b : Bool, truthy (.bool b) = b

def concreteOps : Ops where
  add v w := match v, w with
    | .int a, .int b => .int (a + b)
    | .str a, .str b => .str (a ++ b)
    | _, _ => .nil
  eqv v w := match v, w with
    | .nil, .nil => true
    | .int a, .int b => a == b
    | .bool a, .bool b => a == b
    | .str a, .str b => a == b
    | _, _ => false
  gt v w := match v, w with | .int a, .int b => a > b | _, _ => false
  lt v w := match v, w with | .int a, .int b => a < b | _, _ => false
  truthy v := match v with | .nil => false | .bool b => b | _ => true
  truthy_bool _ := rfl

-- ---------------------------------------------------------------------------
-- 2. Environment and constants (variables are strings, as in Go)
-- ---------------------------------------------------------------------------

def envGet : List (String × Val) → String → Val
  | [], _ => .nil
  | (k, v) :: xs, key => if k = key then v else envGet xs key

def envSet : List (String × Val) → String → Val → List (String × Val)
  | [], k, v => [(k, v)]
  | (k', v') :: xs, k, v =>
      if k' = k then (k, v) :: xs else (k', v') :: envSet xs k v

def constVal (cs : List Val) (i : Nat) : Val := cs.getD i .nil

def constStr (cs : List Val) (i : Nat) : String :=
  match cs.getD i .nil with | .str s => s | _ => ""

-- ---------------------------------------------------------------------------
-- 3. Instruction set (subset covering every fusion in vm_compiler.go)
-- ---------------------------------------------------------------------------

inductive Instr where
  | push (c : Nat)
  | pop
  | add | eq | gt | lt
  | jmp (t : Nat) | jif (t : Nat) | jit (t : Nat)
  | getg (g : Nat) | setg (g : Nat)
  -- fused forms emitted by peephole()
  | addg (g c : Nat)
  | addgg (g1 g2 : Nat)
  | eqgc (g c : Nat) | gtgc (g c : Nat) | ltgc (g c : Nat)
  | fEqJif (g c t : Nat)
  | getgJif (g t : Nat) | getgJit (g t : Nat)
  deriving Inhabited, DecidableEq, Repr

structure State where
  pc : Nat
  stack : List Val          -- head = top of stack (sp = length - 1)
  env : List (String × Val)
  deriving DecidableEq, Repr

inductive Outcome where
  | running (s : State)
  | halted (s : State)      -- pc out of range: normal termination in Go
  | failed (s : State)      -- stack underflow / bad constant
  deriving Repr

def advance (s : State) : State := { s with pc := s.pc + 1 }

def pushV (s : State) (v : Val) : State := { s with stack := v :: s.stack }

def popV (s : State) : Option (Val × State) :=
  match s.stack with
  | [] => none
  | v :: xs => some (v, { s with stack := xs })

def binStep (s : State) (f : Val → Val → Val) : Option State := do
  let (r, s1) ← popV s
  let (l, s2) ← popV s1
  some (pushV (advance s2) (f l r))

-- Execution of one already-fetched instruction: the switch body of
-- runVMMapped in vm.go:37-324.  Split out so proofs only unfold one branch.
def exec (ops : Ops) (cs : List Val) (s : State) : Instr → Outcome
  | .push c => .running (pushV (advance s) (constVal cs c))
  | .pop => match popV s with
      | none => .failed s
      | some (_, s') => .running (advance s')
  | .add => match binStep s ops.add with | none => .failed s | some s' => .running s'
  | .eq => match binStep s (fun a b => .bool (ops.eqv a b)) with
      | none => .failed s | some s' => .running s'
  | .gt => match binStep s (fun a b => .bool (ops.gt a b)) with
      | none => .failed s | some s' => .running s'
  | .lt => match binStep s (fun a b => .bool (ops.lt a b)) with
      | none => .failed s | some s' => .running s'
  | .jmp t => .running { s with pc := t }
  | .jif t => match popV s with
      | none => .failed s
      | some (v, s') => .running (if ops.truthy v then advance s' else { s' with pc := t })
  | .jit t => match popV s with
      | none => .failed s
      | some (v, s') => .running (if ops.truthy v then { s' with pc := t } else advance s')
  | .getg g => .running (pushV (advance s) (envGet s.env (constStr cs g)))
  | .setg g => match s.stack with
      | [] => .failed s
      | v :: _ => .running (advance { s with env := envSet s.env (constStr cs g) v })
  | .addg g c =>
      .running (pushV (advance s) (ops.add (envGet s.env (constStr cs g)) (constVal cs c)))
  | .addgg g1 g2 =>
      .running (pushV (advance s)
        (ops.add (envGet s.env (constStr cs g1)) (envGet s.env (constStr cs g2))))
  | .eqgc g c =>
      .running (pushV (advance s) (.bool (ops.eqv (envGet s.env (constStr cs g)) (constVal cs c))))
  | .gtgc g c =>
      .running (pushV (advance s) (.bool (ops.gt (envGet s.env (constStr cs g)) (constVal cs c))))
  | .ltgc g c =>
      .running (pushV (advance s) (.bool (ops.lt (envGet s.env (constStr cs g)) (constVal cs c))))
  | .fEqJif g c t =>
      let res := ops.eqv (envGet s.env (constStr cs g)) (constVal cs c)
      .running (if res then advance s else { s with pc := t })
  | .getgJif g t =>
      if ops.truthy (envGet s.env (constStr cs g))
      then .running (advance s) else .running { s with pc := t }
  | .getgJit g t =>
      if ops.truthy (envGet s.env (constStr cs g))
      then .running { s with pc := t } else .running (advance s)

-- One fetch-decode-execute step: pc is incremented before dispatch, exactly
-- as in vm.go:34-35.
def step (ops : Ops) (cs : List Val) (p : List Instr) (s : State) : Outcome :=
  if s.pc ≥ p.length then .halted s else
  exec ops cs s (p.getD s.pc (.push 0))

def runN (ops : Ops) (cs : List Val) (p : List Instr) : Nat → State → Outcome
  | 0, s => .running s
  | n + 1, s => match step ops cs p s with
      | .running s' => runN ops cs p n s'
      | other => other

def valueOf (s : State) : Val := match s.stack with | [] => .nil | v :: _ => v

-- ---------------------------------------------------------------------------
-- 4. Structural facts: determinism, forward jumps, termination bound
-- ---------------------------------------------------------------------------

-- step is a pure function, hence deterministic; we state the observable form.
theorem step_deterministic (ops : Ops) (cs : List Val) (p : List Instr) (s s₁ s₂ : State)
    (h₁ : step ops cs p s = .running s₁) (h₂ : step ops cs p s = .running s₂) : s₁ = s₂ := by
  rw [h₁] at h₂
  cases h₂
  rfl

def jTarget : Instr → Option Nat
  | .jmp t | .jif t | .jit t | .fEqJif _ _ t | .getgJif _ t | .getgJit _ t => some t
  | _ => none

-- Compiled programs only jump forward, and never past the end.
def wfJumps (p : List Instr) : Prop :=
  ∀ i, i < p.length → match jTarget (p.getD i (.push 0)) with
    | some t => i < t ∧ t ≤ p.length
    | none => True

-- (Termination is guaranteed because compiled programs only jump forward;
--  the bound is length(program) steps.  We omit that auxiliary proof here and
--  focus on semantic preservation and the cost model below.)

-- ---------------------------------------------------------------------------
-- 5. Peephole fusions are semantics preserving
-- ---------------------------------------------------------------------------

-- Each theorem runs the unfused block for k steps and the fused instruction for
-- one step, starting at the same pc = pre.length.  Stack, environment and the
-- pc.offset (k-1 for forward jumps) are related exactly.

-- Arithmetic facts used by simp when unfolding a few concrete steps.
@[simp] theorem not_add_le_self (n k : Nat) (h : 0 < k) : ¬ n + k ≤ n := by omega
@[simp] theorem not_add_le_add (n a b : Nat) (h : b < a) : ¬ n + a ≤ n + b := by omega

-- All lemmas below compare a fused instruction with the exact instruction
-- sequence that vm_compiler.go:peephole replaces, starting at pc = pre.length.
-- `post` is omitted: only the state at the block boundary matters (the rest
-- of the program is reached through pc and is identical in both programs).

theorem fuse_addg (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g c : Nat) (hpc : s.pc = pre.length) :
    let p := pre ++ [.getg g, .push c, .add]
    let q := pre ++ [.addg g c]
    match runN ops cs p 3 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = pre.length + 3 ∧ b.pc = pre.length + 1 ∧
        a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  simp [runN, step, exec, hpc, Nat.add_assoc,binStep, popV, advance, pushV]

theorem fuse_addgg (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g1 g2 : Nat) (hpc : s.pc = pre.length) :
    let p := pre ++ [.getg g1, .getg g2, .add]
    let q := pre ++ [.addgg g1 g2]
    match runN ops cs p 3 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = pre.length + 3 ∧ b.pc = pre.length + 1 ∧
        a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  simp [runN, step, exec, hpc, Nat.add_assoc,binStep, popV, advance, pushV]

theorem fuse_eqgc (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g c : Nat) (hpc : s.pc = pre.length) :
    let p := pre ++ [.getg g, .push c, .eq]
    let q := pre ++ [.eqgc g c]
    match runN ops cs p 3 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = pre.length + 3 ∧ b.pc = pre.length + 1 ∧
        a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  simp [runN, step, exec, hpc, Nat.add_assoc,binStep, popV, advance, pushV]

theorem fuse_gtgc (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g c : Nat) (hpc : s.pc = pre.length) :
    let p := pre ++ [.getg g, .push c, .gt]
    let q := pre ++ [.gtgc g c]
    match runN ops cs p 3 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = pre.length + 3 ∧ b.pc = pre.length + 1 ∧
        a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  simp [runN, step, exec, hpc, Nat.add_assoc,binStep, popV, advance, pushV]

theorem fuse_ltgc (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g c : Nat) (hpc : s.pc = pre.length) :
    let p := pre ++ [.getg g, .push c, .lt]
    let q := pre ++ [.ltgc g c]
    match runN ops cs p 3 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = pre.length + 3 ∧ b.pc = pre.length + 1 ∧
        a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  simp [runN, step, exec, hpc, Nat.add_assoc,binStep, popV, advance, pushV]

-- 4-instruction fusion: GetGlobal + Push + Equal + JumpIfFalse.
-- oldToNew remaps a target after the block from t to t - 3.
theorem fuse_fEqJif (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g c t : Nat) (hpc : s.pc = pre.length) (ht : t > pre.length + 3) :
    let p := pre ++ [.getg g, .push c, .eq, .jif t]
    let q := pre ++ [.fEqJif g c (t - 3)]
    match runN ops cs p 4 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = b.pc + 3 ∧ a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  by_cases he : ops.eqv (envGet s.env (constStr cs g)) (constVal cs c) = true
  · simp [he, ops.truthy_bool, runN, step, exec, hpc, Nat.add_assoc, binStep,
      popV, advance, pushV]
  · simp [he, ops.truthy_bool, runN, step, exec, hpc, Nat.add_assoc, binStep,
      popV, advance, pushV]
    have hsub : t - 3 + 3 = t := Nat.sub_add_cancel (by omega : 3 ≤ t)
    omega

theorem fuse_getgJif (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g t : Nat) (hpc : s.pc = pre.length) (ht : t > pre.length + 1) :
    let p := pre ++ [.getg g, .jif t]
    let q := pre ++ [.getgJif g (t - 1)]
    match runN ops cs p 2 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = b.pc + 1 ∧ a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  by_cases htval : ops.truthy (envGet s.env (constStr cs g)) = true
  · simp [htval, runN, step, exec, hpc, Nat.add_assoc, popV, advance, pushV,
      ]
  · simp [htval, runN, step, exec, hpc, popV, advance, pushV]
    have hsub : t - 1 + 1 = t := Nat.sub_add_cancel (by omega : 1 ≤ t)
    omega

theorem fuse_getgJit (ops : Ops) (cs : List Val) (pre : List Instr) (s : State)
    (g t : Nat) (hpc : s.pc = pre.length) (ht : t > pre.length + 1) :
    let p := pre ++ [.getg g, .jit t]
    let q := pre ++ [.getgJit g (t - 1)]
    match runN ops cs p 2 s, runN ops cs q 1 s with
    | .running a, .running b =>
        a.pc = b.pc + 1 ∧ a.stack = b.stack ∧ a.env = b.env
    | _, _ => False := by
  dsimp
  have hsub : t - 1 + 1 = t := Nat.sub_add_cancel (by omega : 1 ≤ t)
  by_cases htval : ops.truthy (envGet s.env (constStr cs g)) = true
  · simp [htval, hsub, runN, step, exec, hpc, popV, advance, pushV]
  · simp [htval, runN, step, exec, hpc, popV, advance, pushV]

-- ---------------------------------------------------------------------------
-- 6. Cost model: fusion strictly reduces work whenever dispatch is nonzero
-- ---------------------------------------------------------------------------

structure Cost where
  dispatch : Nat
  push : Nat
  pop : Nat
  lookup : Nat
  binop : Nat

def instrCost (c : Cost) : Instr → Nat
  | .push _ => c.dispatch + c.push
  | .pop => c.dispatch + c.pop
  | .add | .eq | .gt | .lt => c.dispatch + 2 * c.pop + c.push + c.binop
  | .jmp _ => c.dispatch
  | .jif _ | .jit _ => c.dispatch + c.pop
  | .getg _ => c.dispatch + c.lookup + c.push
  | .setg _ => c.dispatch + c.lookup
  | .addg _ _ => c.dispatch + c.lookup + c.push + c.binop
  | .addgg _ _ => c.dispatch + 2 * c.lookup + c.push + c.binop
  | .eqgc _ _ | .gtgc _ _ | .ltgc _ _ => c.dispatch + c.lookup + c.push + c.binop
  | .fEqJif _ _ _ => c.dispatch + c.lookup + c.binop + c.pop
  | .getgJif _ _ | .getgJit _ _ => c.dispatch + c.lookup + c.pop

def programCost (c : Cost) (p : List Instr) : Nat := (p.map (instrCost c)).sum

theorem programCost_append (c : Cost) (p q : List Instr) :
    programCost c (p ++ q) = programCost c p + programCost c q := by
  simp [programCost]

theorem cost_addg (c : Cost) (h : c.dispatch > 0) (g cc : Nat) :
    instrCost c (.addg g cc) <
      instrCost c (.getg g) + instrCost c (.push cc) + instrCost c .add := by
  simp [instrCost]
  omega

theorem cost_addgg (c : Cost) (h : c.dispatch > 0) (g1 g2 : Nat) :
    instrCost c (.addgg g1 g2) <
      instrCost c (.getg g1) + instrCost c (.getg g2) + instrCost c .add := by
  simp [instrCost]
  omega

theorem cost_eqgc (c : Cost) (h : c.dispatch > 0) (g cc : Nat) :
    instrCost c (.eqgc g cc) <
      instrCost c (.getg g) + instrCost c (.push cc) + instrCost c .eq := by
  simp [instrCost]
  omega

theorem cost_fEqJif (c : Cost) (h : c.dispatch > 0) (g cc t : Nat) :
    instrCost c (.fEqJif g cc t) <
      instrCost c (.getg g) + instrCost c (.push cc) +
      instrCost c .eq + instrCost c (.jif t) := by
  simp [instrCost]
  omega

theorem cost_getgJif (c : Cost) (h : c.dispatch > 0) (g t : Nat) :
    instrCost c (.getgJif g t) < instrCost c (.getg g) + instrCost c (.jif t) := by
  simp [instrCost]
  omega

theorem fusion_program_cheaper (c : Cost) (h : c.dispatch > 0)
    (pre post : List Instr) (g cc : Nat) :
    programCost c (pre ++ [.addg g cc] ++ post) <
      programCost c (pre ++ [.getg g, .push cc, .add] ++ post) := by
  repeat rw [programCost_append]
  simp [programCost]
  have hc := cost_addg c h g cc
  omega

end UWasa

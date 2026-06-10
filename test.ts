class A { protected foo = 1; } class B extends A {} function _tcb(this: B) { this.foo; }

from parcs.server import Service, serve
from random import randint


def miller_rabin_iteration(a, r, s):
    n = randint(2, a - 1)
    x = pow(n, s, a)
    if x == 1 or x == a - 1:
        return True
    for _ in range(r - 1):
        x = pow(x, 2, a)
        if x == a - 1:
            return True
    return False


class MillerRabinTest(Service):
    def run(self):
        a, r, s, start, end = self.recv(), self.recv(), self.recv(), self.recv(), self.recv()

        isPrime = True
        for _ in range(start, end):
            if not miller_rabin_iteration(a, r, s):
                isPrime = False
                break

        self.send(isPrime)


serve(MillerRabinTest())
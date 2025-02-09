from parcs.server import Service, serve
import math

def factorize_range(n, start, end):
    factors = []

    if start <= 2:
        while n % 2 == 0:
            factors.append(2)
            n //= 2

    for i in range(max(3, start), end, 2):
         while n % i == 0:
            factors.append(i)
            n //= i
    return factors, n


class Factor(Service):
    def run(self):
        n, start, end = self.recv(), self.recv(), self.recv()

        facts, remaining_n = factorize_range(n, start, end)
        self.send(facts)
        self.send(remaining_n)

serve(Factor())

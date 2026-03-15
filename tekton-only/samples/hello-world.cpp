#include <fstream>
int main() {
  std::ofstream out("output/output.txt");
  out << "Hello from Tekton-only pipeline\n";
  return 0;
}
